package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/SgtCoDFish/rutte"
	"gopkg.in/yaml.v3"
)

type metadataMap map[string]rutte.ManifestMetadata
type replaceMap map[string]string
type descriptionMap map[string]string

const (
	inpath  = "content/en/"
	outpath = "content-out/"

	replacementsFile = "replacements.json"
	descriptionsFile = "descriptions.json"

	metadataFile = "metadata.json"

	verbose = true

	// breakNum can be set to a positive number to get prompted if you want to take a break
	// after breakNum files have been processed
	breakNum = -1
)

var (
	replacementCount int = 0
	filesProcessed   int = 0

	skip bool = false
)

func blockNeedsReplacement(block []byte) bool {
	if bytes.Index(block, []byte("{{%")) != -1 {
		return true
	}

	if bytes.Index(block, []byte("../")) != -1 {
		return true
	}

	if bytes.Index(block, []byte("./")) != -1 {
		return true
	}

	return false
}

func process(knownMetadata metadataMap, knownReplacements replaceMap, knownDescriptions descriptionMap, inpath string, outpath string) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("got an error for %q: %w", path, err)
		}

		if skip {
			return nil
		}

		versionIndependentPath := deriveVersionIndependentPath(path)
		targetPath := strings.Replace(path, inpath, outpath, 1)

		if d.IsDir() {
			// Directory: simply ensure that directories exist at the target path
			err := os.MkdirAll(targetPath, 0o755)
			if err != nil {
				return fmt.Errorf("failed to make dir %q: %w", targetPath, err)
			}

			return nil
		}

		if breakNum > 0 && filesProcessed > 0 && filesProcessed%breakNum == 0 {
			log.Printf("%d files processed; type y to break early, or anything else to continue", breakNum)
			in := bufio.NewReader(os.Stdin)
			response, _ := in.ReadString('\n')
			response = strings.ToLower(response)
			if len(response) >= 1 && response[0] == 'y' {
				skip = true
				return nil
			}
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %q: %w", path, err)
		}

		if !strings.HasSuffix(path, ".md") {
			// Not markdown: simply copy to target path, unchanged
			if verbose {
				log.Printf("copying non-markdown file %q verbatim", path)
			}

			err := os.WriteFile(targetPath, contents, 0o664)
			if err != nil {
				return fmt.Errorf("failed to copy %q verbatim to %q: %w", path, targetPath, err)
			}

			return nil
		}

		if filepath.Base(targetPath) == "_index.md" {
			targetPath = filepath.Join(filepath.Dir(targetPath), "README.md")
		}

		// Separate the header, which is in a block starting and ending with "---"
		parts := bytes.SplitN(contents, []byte("---"), 3)
		if len(parts) != 3 {
			return fmt.Errorf("file %q has malformed header section; only found %d parts but wanted 3", path, len(parts))
		}

		preHeader := parts[0]

		// Defensively check there was nothing before the header; we assume we can reliably parse the header
		// so we need to know if the file isn't in a format we can parse
		if len(bytes.TrimSpace(preHeader)) != 0 {
			return fmt.Errorf("invalid preheader in %q; wanted whitespace only but got %q", path, preHeader)
		}

		header := parts[1]
		inputHeader, err := ParseInputHeader(header)
		if err != nil {
			return fmt.Errorf("failed to parse header block for %q: %w", path, err)
		}

		body := parts[2]

		bodyOut := strings.Builder{}

		scanner := bufio.NewScanner(bytes.NewReader(body))

		// For each line in the file, check if it needs replacement
		// If it does, check if we already know the replacement in our cache and use that if so
		// If we don't, ask the user to give a replacement and then cache it.
		for scanner.Scan() {
			line := append(scanner.Bytes(), byte('\n'))
			lineHash := hashOf(line)

			if !blockNeedsReplacement(line) {
				bodyOut.Write(line)
				continue
			}

			replacementCount += 1

			if verbose {
				log.Printf("need to replace: %q (%s)", strings.TrimSpace(string(line)), lineHash)
			}

			if known, exists := knownReplacements[lineHash]; exists {
				if verbose {
					log.Printf("using cached replacement for %s", lineHash)
				}

				bodyOut.WriteString(known)
			} else {
				newBody, err := promptForEdit(path, line)
				if err != nil {
					return fmt.Errorf("failed to get updated value for line: %w", err)
				}

				_, _ = bodyOut.Write(newBody)

				knownReplacements[lineHash] = string(newBody)

				if verbose {
					log.Printf("added new cached replacement for %s", lineHash)
				}
			}
		}

		completeBody := strings.TrimSpace(bodyOut.String())

		var documentDescription string

		if knownDescription, exists := knownDescriptions[versionIndependentPath]; exists {
			documentDescription = knownDescription
		} else {
			const splitter = "\n>>>>>\n"
			// anything before the first >>>>> will be used as the description
			promptBody := splitter + completeBody
			newDescription, err := promptForEdit(path, []byte(promptBody))
			if err != nil {
				return fmt.Errorf("failed to get description for %q: %w", path, err)
			}

			splitPrompt := bytes.SplitN(newDescription, []byte(splitter), 2)
			if len(splitPrompt) != 2 {
				return fmt.Errorf("invalid description input for %q", path)
			}

			documentDescription = string(splitPrompt[0])
			knownDescriptions[versionIndependentPath] = documentDescription
		}

		headerOut := PageHeader{
			Title:       inputHeader.Title,
			Description: documentDescription,
		}

		output := headerOut.String() + completeBody

		err = os.WriteFile(targetPath, []byte(output), 0o664)
		if err != nil {
			return fmt.Errorf("failed to write %q: %w", targetPath, err)
		}

		knownMetadata[targetPath] = rutte.ManifestMetadata{
			Title:  inputHeader.LinkTitle,
			Weight: inputHeader.Weight,
		}

		filesProcessed += 1

		return nil
	}
}

func createManifests(knownMetadata metadataMap, metadataFiles map[string]*rutte.MetadataFile, inpath string) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		splitPathParts := strings.Split(strings.TrimPrefix(path, inpath), "/")

		docsDir := splitPathParts[0]

		log.Println("docsDir: " + docsDir)

		metadataFile, ok := metadataFiles[docsDir]
		if !ok {
			metadataFiles[docsDir] = rutte.NewMetadataFile()
			metadataFile = metadataFiles[docsDir]
		}

		_, err = rutte.DeepestDir(metadataFile, splitPathParts[1:])
		if err != nil {
			return fmt.Errorf("couldn't find deepest dir for %q: %w", path, err)
		}

		if d.IsDir() {
			// some directories are used locally and in paths but aren't present on the
			// sidebar. We can identify this where a directory doesn't have a README.md
			// e.g. https://cert-manager.io/docs/tutorials/acme/dns-validation/
			_, exists := knownMetadata[filepath.Join(path, "README.md")]
			if !exists {
				// if there's no README, then markdown files under this dir will add
				// themselves as WeightedManifestEntry under the deepest DirManifestEntry
				// they can navigate to
				return nil
			}

			// if there IS a README, add a DirManifestEntry and markdown files in this dir
			// will add themselve to that DirManifestEntry

			return nil
		}

		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		log.Println(metadataFile.Title)

		return nil
	}
}

func main() {
	metadataFiles := make(map[string]*rutte.MetadataFile)

	knownReplacements, err := loadReplacements(replacementsFile)
	if err != nil {
		log.Printf("failed to load %q: %s", replacementsFile, err)
	}

	knownDescriptions, err := loadDescriptions(descriptionsFile)
	if err != nil {
		log.Printf("failed to load %q: %s", descriptionsFile, err)
	}

	knownMetadata, err := loadMetadata(metadataFile)
	if err != nil {
		log.Printf("failed to load %q: %s", metadataFile, err)
	}

	defer writeReplacements(replacementsFile, knownReplacements)
	defer writeDescriptions(descriptionsFile, knownDescriptions)
	defer writeMetadata(metadataFile, knownMetadata)

	if err := filepath.WalkDir(inpath, process(knownMetadata, knownReplacements, knownDescriptions, inpath, outpath)); err != nil {
		log.Printf("failed to process: %s", err.Error())
		os.Exit(1)
	}

	log.Printf("%d replacements total", replacementCount)
	log.Printf("%d unique replacements", len(knownReplacements))

	if err := filepath.WalkDir(outpath, createManifests(knownMetadata, metadataFiles, outpath)); err != nil {
		log.Printf("failed to create manifests: %s", err.Error())
		os.Exit(1)
	}

	exitCode := 0

	for filename, metadataFile := range metadataFiles {
		fullFilename := filepath.Join(outpath, filename, "manifest.json")
		err = os.WriteFile(fullFilename, []byte(metadataFile.String()), 0o664)
		if err != nil {
			log.Printf("failed to write metadata file %q: %s", fullFilename, err)
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}

// util functions

func promptForEdit(originalFilename string, input []byte) ([]byte, error) {
	editorBinary, err := resolveEditor()
	if err != nil {
		return nil, err
	}

	tempDir, err := os.MkdirTemp(os.TempDir(), "rutte-fix-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	defer func() {
		err := os.RemoveAll(tempDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to remove temp dir: %s\n", tempDir)
		}
	}()

	sanitizedOriginal := strings.ReplaceAll(originalFilename, "/", "-") + "-rutte"
	tmpFilename := filepath.Join(tempDir, sanitizedOriginal)

	err = os.WriteFile(tmpFilename, input, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to write %s: %w", tmpFilename, err)
	}

	editorCmd := exec.CommandContext(context.TODO(), editorBinary, tmpFilename)

	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	err = editorCmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to run editor %q: %w", editorBinary, err)
	}

	err = editorCmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for editor %q: %w", editorBinary, err)
	}

	newContents, err := os.ReadFile(tmpFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q: %w", tmpFilename, err)
	}

	return newContents, nil
}

func resolveEditor() (string, error) {
	editorEnv, exists := os.LookupEnv("EDITOR")
	if exists {
		return editorEnv, nil
	}

	for _, name := range []string{"vim", "vi", "nvim"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("couldn't find $EDITOR or any known editor in the $PATH")
}

func loadReplacements(filename string) (replaceMap, error) {
	var out replaceMap

	contents, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return make(replaceMap), nil
		}

		return nil, err
	}

	err = json.Unmarshal(contents, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func writeReplacements(filename string, knownReplacements replaceMap) error {
	out, err := json.MarshalIndent(knownReplacements, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, out, 0o664)
	return err
}

func loadDescriptions(filename string) (descriptionMap, error) {
	var out descriptionMap

	contents, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return make(descriptionMap), nil
		}

		return nil, err
	}

	err = json.Unmarshal(contents, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func writeDescriptions(filename string, knownDescriptions descriptionMap) error {
	out, err := json.MarshalIndent(knownDescriptions, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, out, 0o664)
	return err
}

func loadMetadata(filename string) (metadataMap, error) {
	var out metadataMap

	contents, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return make(metadataMap), nil
		}

		return nil, err
	}

	err = json.Unmarshal(contents, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func writeMetadata(filename string, knownMetadata metadataMap) error {
	out, err := json.MarshalIndent(knownMetadata, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, out, 0o664)
	return err
}

func hashOf(block []byte) string {
	hash := sha256.Sum256(block)
	return hex.EncodeToString(hash[:])
}

// InputHeader is a struct parsed from a hugo-style header
type InputHeader struct {
	Title     string `yaml:"title"`
	LinkTitle string `yaml:"linkTitle"`
	Weight    int64  `yaml:"weight"`
}

func ParseInputHeader(header []byte) (InputHeader, error) {
	var out InputHeader

	err := yaml.Unmarshal(header, &out)
	if err != nil {
		return InputHeader{}, fmt.Errorf("failed to parse header as YAML: %w", err)
	}

	if out.Title == "" {
		return InputHeader{}, fmt.Errorf("missing title in header")
	}

	if out.LinkTitle == "" {
		out.LinkTitle = out.Title
	}

	if out.Weight == 0 {
		out.Weight = 9999
	}

	return out, nil
}

type PageHeader struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description,flow"`
}

func (p *PageHeader) String() string {
	out, _ := yaml.Marshal(p)

	return "---\n" + string(out) + "---\n\n"
}

func deriveVersionIndependentPath(path string) string {
	fullDir, file := filepath.Split(path)

	parts := strings.Split(strings.TrimRight(fullDir, "/"), "/")

	return filepath.Join(parts[len(parts)-1], file)
}

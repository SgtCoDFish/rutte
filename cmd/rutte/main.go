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
)

const inpath = "content/en/"
const outpath = "content-out/"

const verbose = true

var replacementCount int = 0

type replaceMap map[string]string

func blockNeedsReplacement(block []byte) bool {
	if bytes.Index(block, []byte("{{%")) != -1 {
		return true
	}

	if bytes.Index(block, []byte("../")) != -1 {
		return true
	}

	return false
}

func hashOf(block []byte) string {
	hash := sha256.Sum256(block)
	return hex.EncodeToString(hash[:])
}

func process(knownReplacements replaceMap, inpath string, outpath string) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("got an error for %q: %w", path, err)
		}

		targetPath := strings.Replace(path, inpath, outpath, 1)

		if d.IsDir() {
			// Directory: simply ensure that directories exist at the target path
			err := os.MkdirAll(targetPath, 0o755)
			if err != nil {
				return fmt.Errorf("failed to make dir %q: %w", targetPath, err)
			}

			return nil
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
				log.Printf("need to replace: %s (%s)", line, lineHash)
			}

			if known, exists := knownReplacements[lineHash]; exists {
				bodyOut.WriteString(known)
			} else {
				newBody, err := promptForEdit(line)
				if err != nil {
					return fmt.Errorf("failed to get updated value for line: %w", err)
				}

				bodyOut.Write(newBody)

				knownReplacements[lineHash] = string(newBody)
			}
		}

		output := strings.Join([]string{"", string(parts[1]), bodyOut.String()}, "---")

		err = os.WriteFile(targetPath, []byte(output), 0o664)
		if err != nil {
			return fmt.Errorf("failed to write %q: %w", targetPath, err)
		}

		return nil
	}
}

func promptForEdit(input []byte) ([]byte, error) {
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

	tmpFilename := filepath.Join(tempDir, "rutte")

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

	for _, name := range []string{"vim", "vi", "nvim", "nano", "emacs"} {
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
	out, err := json.Marshal(knownReplacements)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, out, 0o664)
	return err
}

func main() {
	replacementsFile := "replacements.json"
	knownReplacements, err := loadReplacements(replacementsFile)
	if err != nil {
		log.Printf("failed to load %q: %s", replacementsFile, err)
	}

	defer writeReplacements(replacementsFile, knownReplacements)

	if err := filepath.WalkDir(inpath, process(knownReplacements, inpath, outpath)); err != nil {
		log.Printf("failed to process: %s", err.Error())
		os.Exit(1)
	}

	log.Printf("%d replacements total", replacementCount)
	log.Printf("%d unique replacements", len(knownReplacements))
}

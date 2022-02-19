package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"os"
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

		scanner := bufio.NewScanner(bytes.NewReader(body))

		// For each line in the file, check if it needs replacement
		// If it does, check if we already know the replacement in our cache and use that if so
		// If we don't, ask the user to give a replacement and then cache it.
		for scanner.Scan() {
			line := scanner.Bytes()
			lineHash := hashOf(line)

			if !blockNeedsReplacement(line) {
				continue
			}

			replacementCount += 1

			if verbose {
				log.Printf("need to replace: %s (%s)", line, lineHash)
			}

			knownReplacements[lineHash] = string(line)
		}

		return nil
	}
}

func main() {
	knownReplacements := make(replaceMap)
	if err := filepath.WalkDir(inpath, process(knownReplacements, inpath, outpath)); err != nil {
		log.Printf("failed to process: %s", err.Error())
		os.Exit(1)
	}

	log.Printf("%d replacements total", replacementCount)
	log.Printf("%d unique replacements", len(knownReplacements))
}

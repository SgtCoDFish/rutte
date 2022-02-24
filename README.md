# rutte

This is a one-off script for migrating the cert-manager website. It takes
old docs which might need some sections replacing, and iteratively prompts
the user for replacements for each section.

If a replacement is known for a piece of text, it'll be cached and used
automatically in the future on the assumption that it's universally applicable.

If a description is known for a file at a given path it'll be used for that file.

After replacing docs + doc headers, the script generates a manifest which acts
as a generator for the sidebar + the docs.

The repo includes checked-in versions of the following files:

- `replacements.json` - cached replacements for links and hugo-style tags
- `descriptions.json` - descriptions, cached for different docs versions where filenames are unchanged
- `metadata.json` - sidebar ordering weights and sidebar titles for generating manifests


## Usage

```bash
go run cmd/rutte/main.go
```

If you remove the json files listed above, you'll be prompted for replacements in your EDTIOR. Get
a coffee first :)

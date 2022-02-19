# rutte

This is a one-off script for migrating the cert-manager website. It takes
old docs which might need some sections replacing, and iteratively prompts
the user for replacements for each section.

If a replacement is known for a piece of text, it'll be cached and used automatically in the future on the assumption that it's universally applicable.

# GoKit CLI Demo

This example demonstrates how to use the GoKit CLI tool for managing internationalization files.

## Setup

1. Make sure you have the GoKit CLI installed:
   ```bash
   go install github.com/kdsmith18542/gokit/cmd/gokit@latest
   ```

2. Navigate to this directory:
   ```bash
   cd examples/cli-demo
   ```

## Examples

### Find Missing Translation Keys

Compare English and Spanish locales to find missing keys:

```bash
gokit i18n find-missing --source=en --target=es --dir=./locales
```

Expected output:
```
Comparing en -> es
Source file: ./locales/en.toml
Target file: ./locales/es.toml

❌ Missing keys in es (3):
  - please
  - yes
  - no
```

### Validate Locale Files

Check all locale files for syntax and completeness:

```bash
gokit i18n validate --dir=./locales
```

Expected output:
```
Validating 3 locale files in ./locales

Validating ./locales/en.toml (en)... ✅ Valid
Validating ./locales/es.toml (es)... ✅ Valid
Validating ./locales/fr.toml (fr)... ⚠️  Warning: 1 empty keys
    - empty_key

✅ All files are valid
```

### Get Help

```bash
gokit i18n help
```

## File Structure

```
locales/
├── en.toml    # English (complete)
├── es.toml    # Spanish (missing some keys)
└── fr.toml    # French (has empty key)
```

## Notes

- The CLI tool currently uses mock implementations for file parsing
- In a real implementation, it would parse TOML files and provide accurate key analysis
- The `extract` command is not yet implemented but will scan source code for translation calls 
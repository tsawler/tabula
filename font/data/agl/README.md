# Glyph mapping sources

`glyphlist.txt`, `zapfdingbats.txt`, and `LICENSE.md` are unmodified Adobe resources from:
https://github.com/adobe-type-tools/agl-aglfn/tree/4036a9ca80a62f64f9de4f7321a9a045ad0ecfd6

The lists contain 4,281 AGL names (81 map to sequences) and 201 Zapf Dingbats names. The Adobe license is also reproduced in the generated Go source.

Regenerate the Go tables offline with `go generate ./font` from the repository root. When updating, replace the three source files together and update the pinned revision here and in the generator. Normal builds do not require generation or network access.

The resolver implements Adobe's suffix, underscore, `uniXXXX...`, and `uXXXX` rules:
https://github.com/adobe-type-tools/agl-specification

Zapf names are only enabled for the Zapf Dingbats font/encoding. Unknown names retain the existing base-encoding fallback. `.notdef` maps to an empty string. `DecodeString` preserves sequences; the single-rune `Decode` returns U+FFFD for a multi-character mapping.

`texGlyphAliases` retains the 13 explicitly tested TeX aliases from the PDF extraction fixes. The semantic aliases `epsilon1`, `follows`, and `lscript` also appear in https://github.com/kohler/lcdf-typetools/blob/master/texglyphlist.txt. Scalable delimiter and summation aliases are approximate shape mappings. The TeX list is not merged wholesale: it contains alternatives, font-specific conflicts, and invalid scalar placeholders. `/ToUnicode` remains authoritative when supplied by the PDF.

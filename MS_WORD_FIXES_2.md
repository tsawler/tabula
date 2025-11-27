# MS Word / Quartz PDF Fixes (Part 2)

## Issues Identified
1.  **Interleaved Text (Overlapping Fragments)**: The PDF generator (Quartz) sometimes places text fragments in a way that they overlap significantly but are drawn in the correct visual order (stream order). For example, "mailing" was drawn as `m` `ai` `l` `i` then `ng ` (with `ng ` placed physically before `l`).
    *   Sorting by X coordinate (standard behavior) reordered this to `maing li` (`ng ` before `l`).
    *   Sorting by stream order (stable sort with tolerance) fixes this.

2.  **Spacing Issues**:
    *   `202 5`: Extra space in year. Caused by aggressive smart spacing or kerning.
    *   `optiona l`: Extra space.
    *   `us. erThe`: `er` drawn inside `. `. Space in `. ` caused split.

## Fixes Implemented

### 1. Stable Sorting with Tolerance
*   Updated `tabula/extractor.go` (`assembleText`) and `tabula/text/extractor.go` (`reorderFragmentsForReading`) to use `sort.SliceStable`.
*   Implemented a "fuzzy" X comparison: if fragments are within 50% of font size (tolerance), they are considered equal in X.
*   This preserves the stream order for overlapping or closely spaced fragments, which is crucial for this PDF where stream order is correct but coordinates are messy.

### 2. CTM Scaling (Previous Fix)
*   Ensured `FontSize` and `Height` are scaled by CTM, so layout analysis works with correct units.

## Verification
*   `mailing` should now appear correctly instead of `maing li`.
*   `head office` should appear correctly (if `o_ice` was due to similar reordering).

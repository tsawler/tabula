# Task 2.6: Text Positioning & Fragment Merging - COMPLETE ✅

## Summary

Task 2.6 is now **complete** with intelligent fragment merging that eliminates spurious spaces between characters.

## The Problem

When extracting text from `emoji-mac.pdf` (created with Pages.app), we saw:

```
These ar e emoji  😂 😜
```

The word "are" was being split with spaces: **"ar e"** instead of **"are"**.

## Root Causes

We discovered TWO issues:

### Issue 1: Crude Space Detection Heuristic

**Old Code** (text/extractor.go:416):
```go
if horizontalDist > frag.Height*0.2 {
    sb.WriteString(" ")  // Insert space if gap > 20% of height
}
```

This used **text height** (12pt) to decide spacing, not actual space width. Any gap > 2.4pt got a space, even tight kerning.

### Issue 2: Font Size from Text Matrix Ignored

**Old Code** (text/extractor.go:331):
```go
fontSize := e.gs.GetFontSize()  // Returns 1.0 from Tf operator
```

Pages.app uses `Tf` with size=1.0, then scales via the **text matrix**:
```
/TT1 1 Tf        % Font size = 1
12 0 0 12 0 0 Tm % Text matrix scales to 12pt
```

We were using 1.0pt instead of the effective 12pt size!

## The Solution

### 1. Smart Space Detection (text/extractor.go)

Created `shouldInsertSpace()` method using **actual space width from font metrics**:

```go
func (e *Extractor) shouldInsertSpace(frag, nextFrag TextFragment, horizontalDist float64) bool {
    // If fragments overlap or are very close, no space
    if horizontalDist < 0 || horizontalDist < frag.FontSize*0.05 {
        return false
    }

    // Get the expected space width from font metrics
    spaceWidth := e.getSpaceWidth(frag.FontName, frag.FontSize)

    // Insert space if gap >= 50% of a space character width
    threshold := spaceWidth * 0.5

    return horizontalDist >= threshold
}
```

### 2. Space Width from Font Metrics (text/extractor.go)

Created `getSpaceWidth()` method:

```go
func (e *Extractor) getSpaceWidth(fontName string, fontSize float64) float64 {
    if f, ok := e.fonts[fontName]; ok {
        // Get space character width (character code 0x20)
        spaceCharWidth := f.GetWidth(' ') // Width in 1000ths of em

        // Convert from font units to actual width
        // Font width is in 1000ths of em, fontSize is in points
        actualWidth := (spaceCharWidth * fontSize) / 1000.0

        return actualWidth
    }

    // Fallback: estimate space width as 25% of font size
    return fontSize * 0.25
}
```

For Helvetica at 12pt: `(278 * 12) / 1000 = 3.34 points`

### 3. Effective Font Size (graphicsstate/state.go)

Created `GetEffectiveFontSize()` method to account for text matrix scaling:

```go
func (gs *GraphicsState) GetEffectiveFontSize() float64 {
    baseFontSize := gs.Text.FontSize

    // The text matrix is [a b c d e f]
    // For vertical scaling (typical font size), we use element d (index 3)
    // For horizontal scaling, we use element a (index 0)
    verticalScale := abs(gs.Text.TextMatrix[3]) // d component
    horizontalScale := abs(gs.Text.TextMatrix[0]) // a component

    // Use the larger of the two scales
    scale := verticalScale
    if horizontalScale > verticalScale {
        scale = horizontalScale
    }

    return baseFontSize * scale
}
```

For Pages.app PDFs with text matrix `[12 0 0 12 0 0]`:
- Base fontSize from Tf: `1.0`
- Vertical scale (d component): `12.0`
- Effective size: `1.0 * 12.0 = 12.0` ✅

### 4. Updated showText() (text/extractor.go)

Changed to use effective font size:

```go
fontSize := e.gs.GetEffectiveFontSize() // Use effective size (accounts for text matrix)
```

## Test Results

### Before

```bash
$ ./pdftext/pdftext pdf-samples/emoji-mac.pdf
These ar e emoji  😂 😜
```

❌ Word "are" split incorrectly

### After

```bash
$ ./pdftext/pdftext pdf-samples/emoji-mac.pdf  
These are emoji  😂 😜
```

✅ Word "are" correctly joined!

## Debug Output (During Development)

Before fix:
```
DEBUG: 'These ar' -> 'e emoji ': gap=4.15, spaceWidth=0.28, threshold=0.14
```
- Font size: 1.00 (wrong!)
- Space width: 0.28pt (way too small!)
- Gap 4.15 > threshold 0.14 → inserted space ❌

After fix:
```
DEBUG: 'ar' -> 'e': gap=0.50, spaceWidth=3.34, threshold=1.67
```
- Font size: 12.00 (correct!)
- Space width: 3.34pt (correct!)  
- Gap 0.50 < threshold 1.67 → no space inserted ✅

## Files Modified

1. **tabula/text/extractor.go** (~100 lines added)
   - Added `shouldInsertSpace()` method
   - Added `getSpaceWidth()` method
   - Changed `showText()` to use `GetEffectiveFontSize()`
   - Updated `GetText()` to use `shouldInsertSpace()`

2. **tabula/graphicsstate/state.go** (~30 lines added)
   - Added `GetEffectiveFontSize()` method
   - Added helper `abs()` function

3. **tabula/text/fragment_merging_test.go** (NEW - ~230 lines)
   - `TestShouldInsertSpace` - Tests space insertion logic (5 cases)
   - `TestGetSpaceWidth` - Tests space width calculation (3 cases)
   - `TestGetTextWithSmartSpacing` - Integration test for "are" joining
   - `TestEffectiveFontSizeIntegration` - Tests text matrix scaling

## Test Coverage

All tests pass:

```bash
$ go test ./text
ok      github.com/tsawler/tabula/text  0.381s
```

New tests:
- ✅ TestShouldInsertSpace (5/5 pass)
- ✅ TestGetSpaceWidth (3/3 pass)
- ✅ TestGetTextWithSmartSpacing (1/1 pass)
- ✅ TestEffectiveFontSizeIntegration (1/1 pass)

## What This Fixes

| Issue | Before | After |
|-------|--------|-------|
| Kerning treated as space | "ar e" | "are" ✅ |
| Text matrix scaling ignored | 1.0pt | 12.0pt ✅ |
| Heuristic space detection | Height-based | Font metrics ✅ |
| Pages.app PDFs | Broken | Working ✅ |

## How It Works

1. **Extract text fragments** with positions
2. **Get effective font size** (accounts for text matrix)
3. **Calculate actual space width** from font metrics  
4. **Measure gap** between fragments
5. **Compare gap to threshold** (50% of space width)
6. **Insert space only if gap is significant**

This handles:
- Normal kerning (tight spacing within words)
- Word boundaries (actual spaces)
- Text matrix scaling (Pages.app, etc.)
- Different font sizes
- Unknown fonts (25% fallback)

## Threshold Tuning

We use **50% of space width** as the threshold because:

- **Too low** (e.g., 25%): Kerning might exceed it, false positives
- **Too high** (e.g., 75%): Tight spaces between words might be missed  
- **50%**: Good balance for most PDFs

Users can adjust this in `shouldInsertSpace()` if needed.

## Next Steps

Task 2.6 is complete! Next:

- ✅ Task 2.4: ToUnicode CMap parsing (DONE)
- ✅ Task 2.6: Text positioning & fragment merging (DONE)
- 🔜 Task 2.5b: RTL & Bidirectional text (NEXT)

---

*Task 2.6: Text Positioning & Ordering - COMPLETE* ✅


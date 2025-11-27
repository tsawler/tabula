# PDF Parsing Deep Dive

This document provides a technical explanation of how PDFs are parsed and how to implement each component.

## PDF File Structure

A PDF file consists of four main parts:

```
%PDF-1.7                          ← Header
...objects...                      ← Body
xref                              ← Cross-reference table
0 6
0000000000 65535 f
0000000015 00000 n
...
trailer                           ← Trailer
<</Size 6/Root 1 0 R>>
startxref
12345                             ← Byte offset of xref
%%EOF
```

## 1. Parsing PDF Objects

### Object Types

PDFs have 8 basic object types:

1. **Boolean**: `true` or `false`
2. **Numeric**: `42` (integer) or `3.14` (real)
3. **String**: `(Hello)` (literal) or `<48656C6C6F>` (hex)
4. **Name**: `/Type`, `/Page`, etc.
5. **Array**: `[1 2 3]`
6. **Dictionary**: `<</Key /Value>>`
7. **Stream**: Dictionary followed by binary data
8. **Null**: `null`

### Parsing Algorithm

```go
func parseObject(reader *bufio.Reader) (Object, error) {
    skipWhitespace(reader)

    // Peek at first byte to determine type
    b := peek(reader)

    switch {
    case b == 't' || b == 'f':
        return parseBoolean(reader)
    case b == '(':
        return parseLiteralString(reader)
    case b == '<':
        next := peek(reader, 1)
        if next == '<' {
            return parseDictionary(reader)
        }
        return parseHexString(reader)
    case b == '[':
        return parseArray(reader)
    case b == '/':
        return parseName(reader)
    case isDigit(b) || b == '-' || b == '+':
        return parseNumber(reader)
    case b == 'n':
        return parseNull(reader)
    }
}
```

### String Parsing

Literal strings support nested parentheses and escape sequences:

```go
func parseLiteralString(reader *bufio.Reader) (string, error) {
    expect(reader, '(')

    var buf bytes.Buffer
    depth := 1

    for depth > 0 {
        b := readByte(reader)

        switch b {
        case '(':
            depth++
            buf.WriteByte(b)
        case ')':
            depth--
            if depth > 0 {
                buf.WriteByte(b)
            }
        case '\\':
            // Handle escape sequences: \n, \r, \t, \\, \(, \), \ddd
            next := readByte(reader)
            switch next {
            case 'n': buf.WriteByte('\n')
            case 'r': buf.WriteByte('\r')
            case 't': buf.WriteByte('\t')
            case '\\', '(', ')': buf.WriteByte(next)
            case '0'-'9':
                // Octal sequence \ddd
                octal := string(next)
                for i := 0; i < 2; i++ {
                    if isOctalDigit(peek(reader)) {
                        octal += string(readByte(reader))
                    }
                }
                value, _ := strconv.ParseInt(octal, 8, 8)
                buf.WriteByte(byte(value))
            }
        default:
            buf.WriteByte(b)
        }
    }

    return buf.String(), nil
}
```

## 2. Cross-Reference Table (XRef)

The XRef table maps object numbers to byte offsets in the file.

### XRef Table Format

```
xref
0 6                     ← Starting object number, count
0000000000 65535 f      ← Free object
0000000015 00000 n      ← Object 1 at byte 15
0000000234 00000 n      ← Object 2 at byte 234
...
```

### Parsing XRef Table

```go
type XRefEntry struct {
    Offset     int64  // Byte offset in file
    Generation int    // Generation number
    InUse      bool   // true = in use, false = free
}

type XRefTable struct {
    entries map[int]*XRefEntry
}

func parseXRefTable(reader *bufio.Reader) (*XRefTable, error) {
    expectKeyword(reader, "xref")

    table := &XRefTable{entries: make(map[int]*XRefEntry)}

    for {
        // Read subsection header
        start, err := readInt(reader)
        if err != nil {
            break // End of xref sections
        }
        count := readInt(reader)

        // Read entries
        for i := 0; i < count; i++ {
            offset := readInt64(reader)
            generation := readInt(reader)
            flag := readToken(reader)

            objNum := start + i
            table.entries[objNum] = &XRefEntry{
                Offset:     offset,
                Generation: generation,
                InUse:      flag == "n",
            }
        }
    }

    return table, nil
}
```

### XRef Streams (PDF 1.5+)

Modern PDFs use XRef streams instead of tables:

```go
type XRefStream struct {
    dict   Dictionary
    data   []byte
    w      [3]int  // Field widths
}

func parseXRefStream(stream *Stream) (*XRefTable, error) {
    // Get field widths from /W array
    w := stream.Dict.Get("W").(Array)
    widths := [3]int{
        int(w[0].(Int)),
        int(w[1].(Int)),
        int(w[2].(Int)),
    }

    data, _ := stream.Decoded()
    table := &XRefTable{entries: make(map[int]*XRefEntry)}

    objNum := 0
    for i := 0; i < len(data); {
        // Read three fields according to widths
        type_ := readBytes(data[i:], widths[0])
        field2 := readBytes(data[i+widths[0]:], widths[1])
        field3 := readBytes(data[i+widths[0]+widths[1]:], widths[2])

        switch type_ {
        case 0: // Free object
            table.entries[objNum] = &XRefEntry{InUse: false}
        case 1: // Normal object
            table.entries[objNum] = &XRefEntry{
                Offset:     field2,
                Generation: int(field3),
                InUse:      true,
            }
        case 2: // Compressed object (in object stream)
            table.entries[objNum] = &XRefEntry{
                Offset:     field2, // Object stream number
                Generation: int(field3), // Index within stream
                InUse:      true,
            }
        }

        i += widths[0] + widths[1] + widths[2]
        objNum++
    }

    return table, nil
}
```

## 3. Content Stream Processing

Content streams contain the actual page content (text, graphics, images).

### Content Stream Operators

```
BT              % Begin text object
/F1 12 Tf       % Set font F1 at 12pt
1 0 0 1 50 750 Tm   % Text matrix (position)
(Hello) Tj      % Show text
ET              % End text object

100 100 m       % Move to (100, 100)
200 100 l       % Line to (200, 100)
S               % Stroke path

q               % Save graphics state
1 0 0 RG        % Set stroke color to red
2 w             % Set line width to 2
Q               % Restore graphics state
```

### Content Stream Parser

```go
type ContentStreamParser struct {
    operators []Operation
    stack     []Object
}

type Operation struct {
    Operator string
    Operands []Object
}

func parseContentStream(data []byte) ([]Operation, error) {
    reader := bufio.NewReader(bytes.NewReader(data))
    parser := NewParser(reader)

    var operations []Operation
    var stack []Object

    for {
        obj, err := parser.ParseObject()
        if err == io.EOF {
            break
        }

        // Check if it's an operator (Name without /)
        if name, ok := obj.(Name); ok {
            // This is an operator
            op := Operation{
                Operator: string(name),
                Operands: stack,
            }
            operations = append(operations, op)
            stack = nil
        } else {
            // This is an operand
            stack = append(stack, obj)
        }
    }

    return operations, nil
}
```

### Graphics State Machine

```go
type GraphicsState struct {
    // Current transformation matrix
    CTM Matrix

    // Color state
    StrokeColor Color
    FillColor   Color

    // Line state
    LineWidth   float64
    LineCap     int
    LineJoin    int

    // Text state
    TextMatrix  Matrix
    TextLineMatrix Matrix
    Font        *Font
    FontSize    float64
    CharSpacing float64
    WordSpacing float64
    Leading     float64
}

type GraphicsStateStack struct {
    states []*GraphicsState
}

func (s *GraphicsStateStack) Push() {
    current := s.Current()
    copy := *current
    s.states = append(s.states, &copy)
}

func (s *GraphicsStateStack) Pop() {
    if len(s.states) > 1 {
        s.states = s.states[:len(s.states)-1]
    }
}

func (s *GraphicsStateStack) Current() *GraphicsState {
    return s.states[len(s.states)-1]
}
```

### Text Extraction

```go
type TextExtractor struct {
    gfxStack    *GraphicsStateStack
    fragments   []TextFragment
    inTextObject bool
}

func (e *TextExtractor) ProcessOperation(op Operation) {
    switch op.Operator {
    case "BT": // Begin text
        e.inTextObject = true
        state := e.gfxStack.Current()
        state.TextMatrix = Identity()
        state.TextLineMatrix = Identity()

    case "ET": // End text
        e.inTextObject = false

    case "Tf": // Set font
        fontName := op.Operands[0].(Name)
        fontSize := float64(op.Operands[1].(Real))
        state := e.gfxStack.Current()
        state.Font = e.getFont(fontName)
        state.FontSize = fontSize

    case "Tm": // Set text matrix
        a := float64(op.Operands[0].(Real))
        b := float64(op.Operands[1].(Real))
        c := float64(op.Operands[2].(Real))
        d := float64(op.Operands[3].(Real))
        tx := float64(op.Operands[4].(Real))
        ty := float64(op.Operands[5].(Real))

        state := e.gfxStack.Current()
        state.TextMatrix = Matrix{a, b, c, d, tx, ty}
        state.TextLineMatrix = state.TextMatrix

    case "Tj": // Show text
        text := string(op.Operands[0].(String))
        e.showText(text)

    case "TJ": // Show text array
        array := op.Operands[0].(Array)
        for _, item := range array {
            switch v := item.(type) {
            case String:
                e.showText(string(v))
            case Int, Real:
                // Adjust position by this amount
                adjustment := float64(v.(Real))
                e.adjustPosition(adjustment)
            }
        }

    case "q": // Save state
        e.gfxStack.Push()

    case "Q": // Restore state
        e.gfxStack.Pop()
    }
}

func (e *TextExtractor) showText(text string) {
    state := e.gfxStack.Current()

    // Decode text using font encoding
    decoded := state.Font.Decode(text)

    // Calculate text position using text matrix and CTM
    tm := state.TextMatrix
    ctm := state.CTM
    finalMatrix := ctm.Multiply(tm)

    position := finalMatrix.Transform(Point{0, 0})

    // Calculate text width
    width := state.Font.StringWidth(decoded, state.FontSize)

    // Create text fragment
    fragment := TextFragment{
        Text:     decoded,
        BBox:     BBox{
            X:      position.X,
            Y:      position.Y,
            Width:  width,
            Height: state.FontSize,
        },
        FontSize: state.FontSize,
        FontName: state.Font.Name,
        Matrix:   finalMatrix,
    }

    e.fragments = append(e.fragments, fragment)

    // Update text matrix (advance position)
    advance := width + float64(len(decoded)) * state.CharSpacing
    state.TextMatrix[4] += advance
}
```

## 4. Font Handling

### Font Types

1. **Type 1** - PostScript fonts
2. **TrueType** - TrueType fonts
3. **Type 3** - User-defined fonts
4. **CID Fonts** - For CJK languages

### Font Descriptor

```go
type Font struct {
    Name     string
    Type     string
    Encoding Encoding
    BaseFont string

    // Metrics
    Ascent  float64
    Descent float64
    CapHeight float64

    // Character widths
    Widths   []float64
    FirstChar int
    LastChar  int

    // For CID fonts
    CMap     *CMap
}

func (f *Font) Decode(raw string) string {
    return f.Encoding.Decode([]byte(raw))
}

func (f *Font) StringWidth(text string, fontSize float64) float64 {
    width := 0.0
    for _, ch := range text {
        charCode := int(ch)
        if charCode >= f.FirstChar && charCode <= f.LastChar {
            width += f.Widths[charCode - f.FirstChar]
        } else {
            width += f.Widths[0] // Default width
        }
    }
    return width * fontSize / 1000.0
}
```

### Encoding

```go
type Encoding interface {
    Decode(data []byte) string
}

type SimpleEncoding struct {
    mapping map[byte]rune
}

func (e *SimpleEncoding) Decode(data []byte) string {
    var result []rune
    for _, b := range data {
        if r, ok := e.mapping[b]; ok {
            result = append(result, r)
        } else {
            result = append(result, rune(b))
        }
    }
    return string(result)
}
```

## 5. Stream Decoding

### Filter Types

PDFs support various compression filters:

- **FlateDecode** - zlib/deflate compression
- **LZWDecode** - LZW compression
- **DCTDecode** - JPEG compression
- **CCITTFaxDecode** - CCITT fax compression
- **JBIG2Decode** - JBIG2 compression
- **JPXDecode** - JPEG 2000 compression

### Stream Decoding

```go
func decodeStream(stream *Stream) ([]byte, error) {
    filter := stream.Dict.Get("Filter")

    if filter == nil {
        return stream.Data, nil
    }

    // Handle single filter
    if name, ok := filter.(Name); ok {
        return applyFilter(string(name), stream.Data, stream.Dict)
    }

    // Handle filter array (multiple filters)
    if array, ok := filter.(Array); ok {
        data := stream.Data
        for _, f := range array {
            name := string(f.(Name))
            data, _ = applyFilter(name, data, stream.Dict)
        }
        return data, nil
    }

    return stream.Data, nil
}

func applyFilter(name string, data []byte, params Dict) ([]byte, error) {
    switch name {
    case "FlateDecode":
        return flateDecompress(data, params)
    case "LZWDecode":
        return lzwDecompress(data, params)
    case "DCTDecode":
        return data, nil // JPEG data is already decoded
    case "ASCIIHexDecode":
        return hexDecode(data)
    case "ASCII85Decode":
        return ascii85Decode(data)
    default:
        return nil, fmt.Errorf("unsupported filter: %s", name)
    }
}

func flateDecompress(data []byte, params Dict) ([]byte, error) {
    reader := flate.NewReader(bytes.NewReader(data))
    defer reader.Close()

    result, err := ioutil.ReadAll(reader)
    if err != nil {
        return nil, err
    }

    // Apply predictor if specified
    if predictor, ok := params.GetInt("Predictor"); ok && predictor > 1 {
        result = applyPredictor(result, params)
    }

    return result, nil
}
```

## 6. Object Streams (PDF 1.5+)

Object streams compress multiple objects together:

```go
type ObjectStream struct {
    stream *Stream
    first  int // Offset of first object
    n      int // Number of objects
}

func (os *ObjectStream) GetObject(index int) (Object, error) {
    // Decode stream
    data, err := os.stream.Decoded()
    if err != nil {
        return nil, err
    }

    // First N integers are object numbers and offsets
    header := data[:os.first]
    objects := data[os.first:]

    // Parse header to get object offsets
    parser := NewParser(bytes.NewReader(header))
    offsets := make(map[int]int)

    for i := 0; i < os.n; i++ {
        objNum, _ := parser.ParseObject()
        offset, _ := parser.ParseObject()
        offsets[int(objNum.(Int))] = int(offset.(Int))
    }

    // Get object at index
    offset := offsets[index]
    objParser := NewParser(bytes.NewReader(objects[offset:]))
    return objParser.ParseObject()
}
```

This guide covers the essential components of PDF parsing. The actual implementation requires handling many edge cases and PDF spec quirks, but this provides the foundation.

// Package contentstream provides parsing of PDF content streams.
//
// Content streams contain the instructions for rendering page content,
// including text display, graphics operations, and image placement.
//
// # Content Stream Operations
//
// PDF content streams consist of operators and their operands:
//
//	parser := contentstream.NewParser(streamData)
//	ops, err := parser.Parse()
//	for _, op := range ops {
//	    fmt.Printf("Operator: %s, Operands: %v\n", op.Operator, op.Operands)
//	}
//
// # Common Operators
//
// Text operators:
//   - BT, ET - Begin/end text object
//   - Tf - Set font and size
//   - Tm - Set text matrix
//   - Tj, TJ - Show text
//   - Td, TD - Move text position
//
// Graphics state operators:
//   - q, Q - Save/restore graphics state
//   - cm - Modify CTM (current transformation matrix)
//   - w - Set line width
//   - J, j - Set line cap/join style
//
// Path operators:
//   - m, l - Move to, line to
//   - re - Rectangle
//   - S, s, f, f* - Stroke and fill paths
//
// # Operand Types
//
// Operands can be any PDF object type:
//   - Numbers (core.Int, core.Real)
//   - Strings (core.String)
//   - Names (core.Name)
//   - Arrays (core.Array)
//   - Dictionaries (core.Dict)
package contentstream

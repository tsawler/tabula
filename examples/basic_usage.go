package main

import (
	"fmt"
	"log"
	"os"

	"github.com/tsawler/tabula"
	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/reader"
	"github.com/tsawler/tabula/tables"
	"github.com/tsawler/tabula/writer"
)

func main() {
	// Example 1: Read a PDF and extract text
	readAndExtractText("input.pdf")

	// Example 2: Extract tables from a PDF
	extractTables("input.pdf")

	// Example 3: Create a new PDF from scratch
	createNewPDF("output.pdf")

	// Example 4: Read PDF and convert to IR for RAG
	convertToIRForRAG("input.pdf")
}

// Example 1: Read a PDF and extract text
func readAndExtractText(filename string) {
	fmt.Println("=== Example 1: Extract Text ===")

	// Open PDF file
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return
	}
	defer file.Close()

	// Create PDF reader
	pdfReader, err := reader.New(file)
	if err != nil {
		log.Printf("Error creating reader: %v", err)
		return
	}

	// Parse document
	doc, err := pdfReader.Parse()
	if err != nil {
		log.Printf("Error parsing PDF: %v", err)
		return
	}

	// Print metadata
	fmt.Printf("Title: %s\n", doc.Metadata.Title)
	fmt.Printf("Author: %s\n", doc.Metadata.Author)
	fmt.Printf("Pages: %d\n", doc.PageCount())

	// Extract text from all pages
	text := doc.ExtractText()
	fmt.Printf("Extracted text:\n%s\n", text)
}

// Example 2: Extract tables from a PDF
func extractTables(filename string) {
	fmt.Println("\n=== Example 2: Extract Tables ===")

	// Open and parse PDF
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return
	}
	defer file.Close()

	pdfReader, err := reader.New(file)
	if err != nil {
		log.Printf("Error creating reader: %v", err)
		return
	}

	doc, err := pdfReader.Parse()
	if err != nil {
		log.Printf("Error parsing PDF: %v", err)
		return
	}

	// Extract tables from all pages
	allTables := doc.ExtractTables()
	fmt.Printf("Found %d tables\n", len(allTables))

	// Process each table
	for i, table := range allTables {
		fmt.Printf("\nTable %d:\n", i+1)
		fmt.Printf("  Rows: %d, Columns: %d\n", table.RowCount(), table.ColCount())
		fmt.Printf("  Confidence: %.2f\n", table.Confidence)
		fmt.Printf("  Has Grid: %v\n", table.HasGrid)

		// Export to markdown
		fmt.Println("\nMarkdown format:")
		fmt.Println(table.ToMarkdown())

		// Export to CSV
		csvFile, err := os.Create(fmt.Sprintf("table_%d.csv", i+1))
		if err == nil {
			csvFile.WriteString(table.ToCSV())
			csvFile.Close()
			fmt.Printf("Saved to table_%d.csv\n", i+1)
		}
	}
}

// Example 3: Create a new PDF from scratch
func createNewPDF(filename string) {
	fmt.Println("\n=== Example 3: Create New PDF ===")

	// Create a new document
	doc := model.NewDocument()
	doc.Metadata.Title = "Sample PDF"
	doc.Metadata.Author = "PDF Library"

	// Create first page
	page := model.NewPage(612, 792) // US Letter size

	// Add a heading
	heading := &model.Heading{
		Text:     "Hello, PDF!",
		Level:    1,
		BBox:     model.NewBBox(50, 700, 512, 50),
		FontSize: 24,
		FontName: "Helvetica-Bold",
	}
	page.AddElement(heading)

	// Add a paragraph
	paragraph := &model.Paragraph{
		Text:      "This is a sample PDF created using the Go PDF library.",
		BBox:      model.NewBBox(50, 600, 512, 80),
		FontSize:  12,
		FontName:  "Helvetica",
		Alignment: model.AlignLeft,
	}
	page.AddElement(paragraph)

	// Add a table
	table := createSampleTable()
	table.BBox = model.NewBBox(50, 400, 512, 150)
	page.AddElement(table)

	doc.AddPage(page)

	// Write PDF to file
	file, err := os.Create(filename)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		return
	}
	defer file.Close()

	pdfWriter := writer.New(file)
	if err := pdfWriter.Write(doc); err != nil {
		log.Printf("Error writing PDF: %v", err)
		return
	}

	fmt.Printf("Created PDF: %s\n", filename)
}

// Helper function to create a sample table
func createSampleTable() *model.Table {
	table := model.NewTable(3, 3)

	// Header row
	table.SetCell(0, 0, model.Cell{Text: "Name", IsHeader: true})
	table.SetCell(0, 1, model.Cell{Text: "Age", IsHeader: true})
	table.SetCell(0, 2, model.Cell{Text: "City", IsHeader: true})

	// Data rows
	table.SetCell(1, 0, model.Cell{Text: "Alice"})
	table.SetCell(1, 1, model.Cell{Text: "30"})
	table.SetCell(1, 2, model.Cell{Text: "New York"})

	table.SetCell(2, 0, model.Cell{Text: "Bob"})
	table.SetCell(2, 1, model.Cell{Text: "25"})
	table.SetCell(2, 2, model.Cell{Text: "San Francisco"})

	table.Confidence = 1.0
	table.HasGrid = true

	return table
}

// Example 4: Convert PDF to IR for RAG
func convertToIRForRAG(filename string) {
	fmt.Println("\n=== Example 4: Convert to IR for RAG ===")

	// Open and parse PDF
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return
	}
	defer file.Close()

	pdfReader, err := reader.New(file)
	if err != nil {
		log.Printf("Error creating reader: %v", err)
		return
	}

	doc, err := pdfReader.Parse()
	if err != nil {
		log.Printf("Error parsing PDF: %v", err)
		return
	}

	// Process each page
	for pageNum, page := range doc.Pages {
		fmt.Printf("\n--- Page %d ---\n", pageNum+1)

		// Process elements in reading order
		for elemIdx, elem := range page.Elements {
			fmt.Printf("\nElement %d: %s\n", elemIdx+1, elem.Type())
			fmt.Printf("  BBox: (%.1f, %.1f, %.1f, %.1f)\n",
				elem.BoundingBox().X,
				elem.BoundingBox().Y,
				elem.BoundingBox().Width,
				elem.BoundingBox().Height)

			// Extract text content
			if textElem, ok := elem.(model.TextElement); ok {
				text := textElem.GetText()
				if len(text) > 100 {
					text = text[:100] + "..."
				}
				fmt.Printf("  Text: %s\n", text)
			}

			// Special handling for tables
			if table, ok := elem.(*model.Table); ok {
				fmt.Printf("  Table: %dx%d (confidence: %.2f)\n",
					table.RowCount(), table.ColCount(), table.Confidence)

				// For RAG, you might want to serialize tables specially
				// E.g., as markdown or structured JSON
				ragTableRepresentation := serializeTableForRAG(table)
				fmt.Printf("  RAG representation: %s\n", ragTableRepresentation)
			}
		}
	}
}

// Serialize table in a format suitable for RAG ingestion
func serializeTableForRAG(table *model.Table) string {
	// Option 1: Markdown format (good for most LLMs)
	return table.ToMarkdown()

	// Option 2: Plain text with clear structure
	// Option 3: JSON format
	// etc.
}

// Advanced: Configure table detection
func configureTableDetection() {
	fmt.Println("\n=== Configure Table Detection ===")

	// Get the geometric detector
	detector := tables.GetDetector("geometric")
	if detector == nil {
		log.Println("Geometric detector not found")
		return
	}

	// Configure with custom settings
	config := tables.Config{
		MinRows:            3,
		MinCols:            2,
		MinConfidence:      0.7,
		UseLines:           true,
		UseWhitespace:      true,
		MaxCellGap:         10.0,
		AlignmentTolerance: 3.0,
		DetectMergedCells:  true,
	}

	if err := detector.Configure(config); err != nil {
		log.Printf("Error configuring detector: %v", err)
		return
	}

	fmt.Println("Table detector configured successfully")
}

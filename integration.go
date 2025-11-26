// integration.go provides methods to integrate layout analysis with the Document/Page models
package tabula

import (
	"github.com/tsawler/tabula/layout"
	"github.com/tsawler/tabula/model"
	"github.com/tsawler/tabula/reader"
	"github.com/tsawler/tabula/text"
)

// AnalyzeDocument performs layout analysis on all pages of a document and populates
// the Layout field of each page. This enables access to detected headings, lists,
// paragraphs, columns, and other structural elements.
//
// Example:
//
//	doc, err := tabula.AnalyzeDocument("document.pdf")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, page := range doc.Pages {
//	    fmt.Printf("Page %d: %d headings, %d paragraphs\n",
//	        page.Number, len(page.GetHeadings()), len(page.GetParagraphs()))
//	}
func AnalyzeDocument(path string) (*model.Document, error) {
	return AnalyzeDocumentWithConfig(path, layout.DefaultAnalyzerConfig())
}

// AnalyzeDocumentWithConfig performs layout analysis with custom configuration
func AnalyzeDocumentWithConfig(path string, config layout.AnalyzerConfig) (*model.Document, error) {
	r, err := reader.Open(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	doc := model.NewDocument()

	// Get page count
	pageCount, err := r.PageCount()
	if err != nil {
		return nil, err
	}

	// Create analyzer
	analyzer := layout.NewAnalyzerWithConfig(config)

	// Collect all pages for header/footer detection
	var allPageFragments []layout.PageFragments
	for i := 0; i < pageCount; i++ {
		page, err := r.GetPage(i)
		if err != nil {
			continue
		}
		fragments, err := r.ExtractTextFragments(page)
		if err != nil {
			continue
		}
		width, _ := page.Width()
		height, _ := page.Height()
		allPageFragments = append(allPageFragments, layout.PageFragments{
			PageIndex:  i,
			PageWidth:  width,
			PageHeight: height,
			Fragments:  fragments,
		})
	}

	// Detect headers/footers across all pages
	var headerFooterResult *layout.HeaderFooterResult
	if len(allPageFragments) >= 2 {
		hfDetector := layout.NewHeaderFooterDetector()
		headerFooterResult = hfDetector.Detect(allPageFragments)
	}

	// Process each page
	for i := 0; i < pageCount; i++ {
		pdfPage, err := r.GetPage(i)
		if err != nil {
			continue
		}

		width, _ := pdfPage.Width()
		height, _ := pdfPage.Height()

		modelPage := model.NewPage(width, height)
		modelPage.Number = i + 1

		// Extract fragments
		fragments, err := r.ExtractTextFragments(pdfPage)
		if err != nil {
			doc.AddPage(modelPage)
			continue
		}

		// Store raw text fragments
		modelPage.RawText = convertToModelFragments(fragments)

		// Filter headers/footers if detected
		if headerFooterResult != nil {
			fragments = headerFooterResult.FilterFragments(i, fragments, height)
		}

		// Perform layout analysis
		result := analyzer.Analyze(fragments, width, height)

		// Populate page layout
		modelPage.Layout = convertAnalysisToLayout(result, headerFooterResult, i, height)

		// Populate elements from analysis
		modelPage.Elements = convertLayoutElementsToModelElements(result.Elements)

		doc.AddPage(modelPage)
	}

	return doc, nil
}

// PopulatePageLayout performs layout analysis on a single page and populates its Layout field
func PopulatePageLayout(page *model.Page, fragments []text.TextFragment) {
	PopulatePageLayoutWithConfig(page, fragments, layout.DefaultAnalyzerConfig())
}

// PopulatePageLayoutWithConfig performs layout analysis with custom configuration
func PopulatePageLayoutWithConfig(page *model.Page, fragments []text.TextFragment, config layout.AnalyzerConfig) {
	analyzer := layout.NewAnalyzerWithConfig(config)
	result := analyzer.Analyze(fragments, page.Width, page.Height)

	page.Layout = convertAnalysisToLayout(result, nil, page.Number-1, page.Height)
	page.Elements = convertLayoutElementsToModelElements(result.Elements)
}

// convertAnalysisToLayout converts layout.AnalysisResult to model.PageLayout
func convertAnalysisToLayout(result *layout.AnalysisResult, hfResult *layout.HeaderFooterResult, pageIndex int, pageHeight float64) *model.PageLayout {
	pl := &model.PageLayout{
		ColumnCount: result.Stats.ColumnCount,
		Stats: model.LayoutStats{
			FragmentCount:  result.Stats.FragmentCount,
			LineCount:      result.Stats.LineCount,
			BlockCount:     result.Stats.BlockCount,
			ParagraphCount: result.Stats.ParagraphCount,
			HeadingCount:   result.Stats.HeadingCount,
			ListCount:      result.Stats.ListCount,
		},
	}

	// Convert columns
	if result.Columns != nil {
		for i, col := range result.Columns.Columns {
			pl.Columns = append(pl.Columns, model.ColumnInfo{
				Index: i,
				Left:  col.BBox.X,
				Right: col.BBox.X + col.BBox.Width,
				Width: col.BBox.Width,
				BBox:  col.BBox, // model.BBox is compatible
			})
		}
	}

	// Convert blocks
	if result.Blocks != nil {
		for i, block := range result.Blocks.Blocks {
			pl.TextBlocks = append(pl.TextBlocks, model.BlockInfo{
				Index:     i,
				BBox:      block.BBox,
				LineCount: block.LineCount(),
				Text:      block.GetText(),
				Column:    -1, // Block doesn't track column index
				FontSize:  block.AverageFontSize(),
				Alignment: model.AlignmentUnknown, // Block doesn't track alignment
			})
		}
	}

	// Convert paragraphs
	if result.Paragraphs != nil {
		for i, para := range result.Paragraphs.Paragraphs {
			pl.Paragraphs = append(pl.Paragraphs, model.ParagraphInfo{
				Index:      i,
				BBox:       para.BBox,
				Text:       para.Text,
				FontSize:   para.AverageFontSize,
				FontName:   "", // Paragraph doesn't track font name
				LineCount:  len(para.Lines),
				Alignment:  convertLineAlignment(para.Alignment),
				FirstLine:  para.FirstLineIndent,
				LineHeight: para.LineSpacing,
			})
		}
	}

	// Convert lines
	if result.Lines != nil {
		for i, line := range result.Lines.Lines {
			pl.Lines = append(pl.Lines, model.LineInfo{
				Index:     i,
				BBox:      line.BBox,
				Text:      line.Text,
				FontSize:  line.AverageFontSize,
				Alignment: convertLineAlignment(line.Alignment),
				IsIndent:  line.Indentation > 0,
			})
		}
	}

	// Convert headings
	if result.Headings != nil {
		for _, heading := range result.Headings.Headings {
			pl.Headings = append(pl.Headings, model.HeadingInfo{
				Level:      int(heading.Level), // HeadingLevel is int type
				Text:       heading.Text,
				BBox:       heading.BBox,
				FontSize:   heading.FontSize,
				FontName:   "", // Heading doesn't track font name
				Confidence: heading.Confidence,
			})
		}
	}

	// Convert lists
	if result.Lists != nil {
		for _, list := range result.Lists.Lists {
			// Check if list has nested items
			hasNested := false
			startValue := 1
			for _, item := range list.Items {
				if len(item.Children) > 0 {
					hasNested = true
				}
				if item.Index == 0 && item.Number > 0 {
					startValue = item.Number
				}
			}

			listInfo := model.ListInfo{
				Type:       convertListType(list.Type),
				BBox:       list.BBox,
				Nested:     hasNested,
				StartValue: startValue,
			}
			for _, item := range list.Items {
				listInfo.Items = append(listInfo.Items, model.ListItem{
					Text:   item.Text,
					BBox:   item.BBox,
					Bullet: item.Prefix,
					Level:  item.Level,
				})
			}
			pl.Lists = append(pl.Lists, listInfo)
		}
	}

	// Set header/footer info if available
	// A region is considered "repeating" if it appears on multiple pages
	if hfResult != nil {
		if len(hfResult.Headers) > 0 && len(hfResult.Headers[0].PageIndices) >= 2 {
			pl.HasHeader = true
			pl.HeaderHeight = hfResult.Headers[0].BBox.Y + hfResult.Headers[0].BBox.Height
		}
		if len(hfResult.Footers) > 0 && len(hfResult.Footers[0].PageIndices) >= 2 {
			pl.HasFooter = true
			pl.FooterHeight = pageHeight - hfResult.Footers[0].BBox.Y
		}
	}

	return pl
}

// convertLayoutElementsToModelElements converts layout.LayoutElement slice to model.Element slice
func convertLayoutElementsToModelElements(elements []layout.LayoutElement) []model.Element {
	result := make([]model.Element, 0, len(elements))
	for _, elem := range elements {
		modelElem := elem.ToModelElement()
		if modelElem != nil {
			result = append(result, modelElem)
		}
	}
	return result
}

// convertToModelFragments converts text.TextFragment slice to model.TextFragment slice
func convertToModelFragments(fragments []text.TextFragment) []model.TextFragment {
	result := make([]model.TextFragment, len(fragments))
	for i, f := range fragments {
		result[i] = model.TextFragment{
			Text:     f.Text,
			BBox:     model.BBox{X: f.X, Y: f.Y, Width: f.Width, Height: f.Height},
			FontSize: f.FontSize,
			FontName: f.FontName,
		}
	}
	return result
}

// convertLineAlignment converts layout.LineAlignment to model.Alignment
func convertLineAlignment(align layout.LineAlignment) model.Alignment {
	switch align {
	case layout.AlignLeft:
		return model.AlignmentLeft
	case layout.AlignCenter:
		return model.AlignmentCenter
	case layout.AlignRight:
		return model.AlignmentRight
	case layout.AlignJustified:
		return model.AlignmentJustified
	default:
		return model.AlignmentUnknown
	}
}

// convertListType converts layout list type to model list type
func convertListType(lt layout.ListType) model.ListType {
	switch lt {
	case layout.ListTypeBullet:
		return model.ListTypeBullet
	case layout.ListTypeNumbered:
		return model.ListTypeNumbered
	case layout.ListTypeLettered:
		return model.ListTypeLettered
	case layout.ListTypeRoman:
		return model.ListTypeRoman
	case layout.ListTypeCheckbox:
		return model.ListTypeCheckbox
	default:
		return model.ListTypeUnknown
	}
}

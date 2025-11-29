package epubdoc

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

// DRM-related errors.
var (
	ErrDRMProtected = errors.New("epub: DRM-protected content cannot be processed")
)

// encryptionXML represents the structure of META-INF/encryption.xml.
type encryptionXML struct {
	XMLName       xml.Name        `xml:"encryption"`
	EncryptedData []encryptedData `xml:"EncryptedData"`
}

type encryptedData struct {
	EncryptionMethod encryptionMethod `xml:"EncryptionMethod"`
	CipherData       cipherData       `xml:"CipherData"`
}

type encryptionMethod struct {
	Algorithm string `xml:"Algorithm,attr"`
}

type cipherData struct {
	CipherReference cipherReference `xml:"CipherReference"`
}

type cipherReference struct {
	URI string `xml:"URI,attr"`
}

// checkForDRM checks if the EPUB has DRM protection.
// Returns ErrDRMProtected if DRM is detected.
func checkForDRM(zr *zip.Reader) error {
	for _, f := range zr.File {
		switch f.Name {
		case "META-INF/rights.xml":
			// Adobe ADEPT DRM indicator - always reject
			return ErrDRMProtected

		case "META-INF/encryption.xml":
			// Need to parse to check if content files are encrypted
			// (font obfuscation is OK, content encryption is not)
			encrypted, err := hasEncryptedContent(f)
			if err != nil {
				// If we can't parse it, assume it's DRM
				return ErrDRMProtected
			}
			if encrypted {
				return ErrDRMProtected
			}
		}
	}
	return nil
}

// hasEncryptedContent parses encryption.xml and checks if any content files
// (XHTML, HTML) are encrypted. Font obfuscation is allowed.
func hasEncryptedContent(f *zip.File) (bool, error) {
	rc, err := f.Open()
	if err != nil {
		return false, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return false, err
	}

	var enc encryptionXML
	if err := xml.Unmarshal(data, &enc); err != nil {
		return false, err
	}

	// Check each encrypted resource
	for _, ed := range enc.EncryptedData {
		uri := strings.ToLower(ed.CipherData.CipherReference.URI)

		// Font obfuscation algorithms are OK
		algo := ed.EncryptionMethod.Algorithm
		if isFontObfuscation(algo) {
			continue
		}

		// Check if this is a content file (not a font or image)
		if isContentFile(uri) {
			return true, nil
		}
	}

	return false, nil
}

// isFontObfuscation returns true if the algorithm is a font obfuscation method.
// Font obfuscation is not DRM - it's just to prevent casual font extraction.
func isFontObfuscation(algorithm string) bool {
	// Adobe font obfuscation
	if strings.Contains(algorithm, "adobe.com") && strings.Contains(algorithm, "obfuscation") {
		return true
	}
	// IDPF font obfuscation
	if strings.Contains(algorithm, "idpf.org") && strings.Contains(algorithm, "obfuscation") {
		return true
	}
	return false
}

// isContentFile returns true if the URI refers to a content file that would
// indicate DRM if encrypted.
func isContentFile(uri string) bool {
	uri = strings.ToLower(uri)

	// Content files
	if strings.HasSuffix(uri, ".xhtml") ||
		strings.HasSuffix(uri, ".html") ||
		strings.HasSuffix(uri, ".htm") ||
		strings.HasSuffix(uri, ".xml") {
		return true
	}

	// CSS could also indicate DRM
	if strings.HasSuffix(uri, ".css") {
		return true
	}

	return false
}

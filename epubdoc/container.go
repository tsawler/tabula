package epubdoc

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io"
)

// Container-related errors.
var (
	ErrNoContainer      = errors.New("epub: missing META-INF/container.xml")
	ErrInvalidContainer = errors.New("epub: invalid container.xml")
	ErrNoRootfile       = errors.New("epub: no rootfile found in container.xml")
)

// containerXML represents the structure of META-INF/container.xml.
type containerXML struct {
	XMLName   xml.Name  `xml:"container"`
	Version   string    `xml:"version,attr"`
	Rootfiles rootfiles `xml:"rootfiles"`
}

type rootfiles struct {
	Rootfile []rootfile `xml:"rootfile"`
}

type rootfile struct {
	FullPath  string `xml:"full-path,attr"`
	MediaType string `xml:"media-type,attr"`
}

// parseContainer parses META-INF/container.xml and returns the path to the OPF file.
func parseContainer(zr *zip.Reader) (string, error) {
	// Find container.xml
	var containerFile *zip.File
	for _, f := range zr.File {
		if f.Name == "META-INF/container.xml" {
			containerFile = f
			break
		}
	}

	if containerFile == nil {
		return "", ErrNoContainer
	}

	// Read and parse
	rc, err := containerFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	var container containerXML
	if err := xml.Unmarshal(data, &container); err != nil {
		return "", ErrInvalidContainer
	}

	// Find the OPF rootfile
	for _, rf := range container.Rootfiles.Rootfile {
		if rf.MediaType == "application/oebps-package+xml" || rf.MediaType == "" {
			if rf.FullPath != "" {
				return rf.FullPath, nil
			}
		}
	}

	// If no media-type match, just return the first one
	if len(container.Rootfiles.Rootfile) > 0 {
		return container.Rootfiles.Rootfile[0].FullPath, nil
	}

	return "", ErrNoRootfile
}

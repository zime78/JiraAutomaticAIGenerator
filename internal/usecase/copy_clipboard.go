package usecase

import "jira-ai-generator/internal/port"

// CopyToClipboardUseCase handles copying content to clipboard
type CopyToClipboardUseCase struct {
	docGenerator port.DocumentGenerator
	clipboard    port.Clipboard
}

// NewCopyToClipboardUseCase creates a new CopyToClipboardUseCase
func NewCopyToClipboardUseCase(docGenerator port.DocumentGenerator, clipboard port.Clipboard) *CopyToClipboardUseCase {
	return &CopyToClipboardUseCase{
		docGenerator: docGenerator,
		clipboard:    clipboard,
	}
}

// Execute copies the document content to clipboard
func (uc *CopyToClipboardUseCase) Execute(doc interface{ GetContent() string }) {
	content := doc.GetContent()
	uc.clipboard.SetContent(content)
}

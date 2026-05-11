package prompt

import "fmt"

// Loader provides access to prompt templates based on language.
type Loader interface {
	GetTemplate(templateName string, lang string) (string, error)
}

type staticLoader struct{}

// NewStaticLoader returns a simple prompt loader.
func NewStaticLoader() Loader {
	return &staticLoader{}
}

func (l *staticLoader) GetTemplate(templateName string, lang string) (string, error) {
	if lang == "" {
		lang = "en"
	}

	var tpl map[string]string
	switch templateName {
	case "summary_entry_chats":
		tpl = SummaryEntryChatsTemplate
	case "extract_profile":
		tpl = ExtractProfileTemplate
	case "merge_profile_yolo":
		tpl = MergeProfileYoloTemplate
	default:
		return "", fmt.Errorf("unknown template: %s", templateName)
	}

	content, ok := tpl[lang]
	if !ok {
		return tpl["en"], nil // Fallback to English
	}
	return content, nil
}

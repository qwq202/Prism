package article

import (
	"chat/globals"
	"chat/utils"
	"fmt"

	"github.com/lukasjarosch/go-docx"
)

func GenerateDocxFile(target, title, content string) error {
	data := docx.PlaceholderMap{
		"title":   title,
		"content": content,
	}

	doc, err := docx.Open("addition/article/template.docx")
	if err != nil {
		return err
	}

	if err := doc.ReplaceAll(data); err != nil {
		return err
	}

	if err := doc.WriteToFile(target); err != nil {
		return err
	}

	return nil
}

func articleDocxPath(hash, title string) (string, error) {
	return utils.SafeJoin("storage/article/data", hash, utils.SafeFileName(title, "article")+".docx")
}

func CreateArticleFile(hash, title, content string) string {
	target, err := articleDocxPath(hash, title)
	if err != nil {
		globals.Debug(fmt.Sprintf("[article] unsafe article filename %q: %s", title, err.Error()))
		return ""
	}

	utils.FileDirSafe(target)
	if err := GenerateDocxFile(target, title, content); err != nil {
		globals.Debug(fmt.Sprintf("[article] error during generate article %s: %s", title, err.Error()))
	}

	return target
}

package generation

import (
	"chat/globals"
	"chat/utils"
	"fmt"
	"time"
)

func GetFolder(hash string) string {
	return fmt.Sprintf("storage/generation/data/%s", hash)
}

func GetFolderByHash(model string, prompt string) (string, string) {
	hash := utils.Sha2Encrypt(model + prompt + time.Now().Format("2006-01-02 15:04:05"))
	return hash, GetFolder(hash)
}

func GenerateProject(path string, instance ProjectResult) bool {
	for name, data := range instance.Result {
		current, err := utils.SafeJoin(path, name)
		if err != nil {
			globals.Debug(fmt.Sprintf("[generation] reject unsafe project path %q: %s", name, err.Error()))
			return false
		}

		switch content := data.(type) {
		case string:
			if utils.WriteFile(current, content, true) != nil {
				return false
			}
		case map[string]interface{}:
			if !GenerateProject(current, ProjectResult{Result: content}) {
				return false
			}
		default:
			globals.Debug(fmt.Sprintf("[generation] reject unsupported project node for %q", name))
			return false
		}
	}
	return true
}

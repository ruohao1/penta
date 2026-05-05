package cli

import "github.com/ruohao1/penta/internal/ids"

func generateID() string {
	return ids.Token()
}

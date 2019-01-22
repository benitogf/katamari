package samo

import (
	"encoding/base64"
	"strings"

	"gopkg.in/godo.v2/glob"
)

func (app *Server) validKey(key string, separator string) bool {
	// https://stackoverflow.com/a/26792316/6582356
	return !strings.Contains(key, separator+separator)
}

func (app *Server) extractMoIndex(index string, separator string) string {
	return index[strings.LastIndexAny(index, separator)+1:]
}

func (app *Server) isMO(key string, index string, separator string) bool {
	moIndex := strings.Split(strings.Replace(index, key+separator, "", 1), separator)
	return len(moIndex) == 1 && moIndex[0] != key
}

func (app *Server) extractNonNil(event map[string]interface{}, field string) string {
	data := ""
	if event[field] != nil {
		data = event[field].(string)
	}

	return data
}

func (app *Server) generateRouteRegex(separator string) string {
	return "[a-zA-Z\\d][a-zA-Z\\d\\" + separator + "]+[a-zA-Z\\d]"
}

func (app *Server) checkArchetype(key string, data string, archetypes Archetypes) bool {
	found := ""
	for ar := range archetypes {
		if glob.Globexp(ar).MatchString(key) {
			found = ar
		}
	}
	if found != "" {
		return archetypes[found](data)
	}

	return true
}

func (app *Server) encodeData(raw []byte) string {
	data := ""
	if len(raw) > 0 {
		data = base64.StdEncoding.EncodeToString(raw)
	}

	return data
}

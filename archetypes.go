package samo

import "gopkg.in/godo.v2/glob"

// Archetype : function to check proper key->data covalent bond
type Archetype func(index string, data string) bool

// Archetypes : a map that allows structure and content formalization of key->data
type Archetypes map[string]Archetype

func (arch Archetypes) check(key string, index string, data string, static bool) bool {
	found := ""
	for ar := range arch {
		if glob.Globexp(ar).MatchString(key) {
			found = ar
		}
	}
	if found != "" {
		return arch[found](index, data)
	}

	return !static
}

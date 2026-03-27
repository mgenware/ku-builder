package util

/**
* Why lib type cache?
* LibType has to be passed by caller during setup phase. During building phase, we need to pass lib type as env var
* (some libs may use this to determine how to build). So we cache the lib type during setup phase and retrieve it during build phase.
* K: build dir generated during setup phase, V: lib type.
**/
var kuLibTypeCache = make(map[string]string)

func CacheKuLibType(buildDir string, libType string) {
	kuLibTypeCache[buildDir] = libType
}

func GetCachedKuLibType(buildDir string) (string, bool) {
	libType, ok := kuLibTypeCache[buildDir]
	return libType, ok
}

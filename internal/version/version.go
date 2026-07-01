package version

var (
	ver    = "dev"
	commit = "none"
	date   = "unknown"
)

func Info() string {
	return ver + " (" + commit + ") built " + date
}

func Set(v, c, d string) {
	ver = v
	commit = c
	date = d
}

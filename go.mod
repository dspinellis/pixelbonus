module pixelbonus

go 1.15

require (
	github.com/mattn/go-sqlite3 v0.0.0
	golang.org/x/crypto v0.0.0
)

replace github.com/mattn/go-sqlite3 => ./third_party/github.com/mattn/go-sqlite3
replace golang.org/x/crypto => ./third_party/golang.org/x/crypto

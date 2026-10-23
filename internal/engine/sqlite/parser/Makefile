GO ?= go

sqlite_parser.go: SQLiteLexer.g4 SQLiteParser.g4 antlr-4.13.1-complete.jar Makefile ../../../../scripts/fix-antlr-exits/main.go
	java -jar antlr-4.13.1-complete.jar -Dlanguage=Go SQLiteLexer.g4 SQLiteParser.g4
	$(GO) run ../../../../scripts/fix-antlr-exits $@

antlr-4.13.1-complete.jar:
	curl -O https://www.antlr.org/download/antlr-4.13.1-complete.jar



default: mo
	go run .

mo:
	msgfmt -c -v po/default.pot -o mo/en_GB.utf8/LC_MESSAGES/default.mo

.PHONY: mo default

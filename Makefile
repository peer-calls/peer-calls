export PATH := node_modules/.bin:$(PATH)
SHELL=/bin/bash

.PHONY: start
start:

	chastifol [ make watchify ] [ make sassify ] [ make server ]

.PHONY: build
build: sass js

.PHONY: watchify
watchify:

	watchify -d -v -t babelify ./src/client/index.js -o ./build/index.js

.PHONY: js
js:

	browserify -d -v -t babelify ./src/client/index.js -o ./build/index.js

.PHONY: sass
sass:

	node-sass ./src/scss/style.scss -o ./build/

.PHONY: sassify
sassify: sass

	node-sass --watch ./src/scss/style.scss -o ./build/

.PHONY: lint
lint:

	eslint src/

.PHONY: lint-fix
lint-fix:

	eslint --fix src/

.PHONY: test
test:

	jest --forceExit

.PHONY: testify
testify:

	jest --watch

.PHONY: coverage
coverage:

	jest --coverage --forceExit

.PHONY: server
server:

	nodemon --ignore src/client ./src/index.js

.PHONY: clean
clean:

	rm -rf dist/

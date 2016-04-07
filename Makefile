export PATH := node_modules/.bin:$(PATH)
SHELL=/bin/bash

.PHONY: build
build:

	mkdir -p dist/client dist/css

	browserify -t babelify ./src/client/index.js | uglifyjs --comments -o ./dist/client/index.js

	lessc ./src/less/main.less ./dist/css/main.css

	cp -v ./src/index.js ./dist/index.js
	cp -rv ./src/server ./dist/
	cp -rv ./src/less/fonts ./dist/css/
	cp -rv ./src/views ./dist/
	cp -rv ./src/res ./dist/

.PHONY: lint
lint:

	eslint src/

.PHONY: test
test:

	jest --verbose

.PHONY: testify
testify:

	jest --watch

.PHONY: coverage
coverage:

	jest --coverage

.PHONY: run
run:

	node ./src/index.js

.PHONY: clean
clean:

	rm -rf dist/

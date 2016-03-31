export PATH := node_modules/.bin:$(PATH)
SHELL=/bin/bash

.PHONY: build
build: clean

	eslint src/

	jest --coverage

	mkdir -p dist/js dist/less

	browserify -t babelify ./src/js/index.js -o ./dist/js/index.js

	lessc ./src/less/main.less ./dist/less/main.css

	cp ./src/index.js ./dist/index.js
	cp -r ./src/server ./dist/server
	cp -r ./src/less/fonts ./dist/less/fonts
	cp -r ./src/views ./dist/views
	cp -r ./src/res ./dist/res

.PHONY: test
test:

	jest

.PHONY: run
run:

	node ./src/index.js

.PHONY: clean
clean:

	rm -rf dist/

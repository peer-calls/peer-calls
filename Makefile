export PATH := node_modules/.bin:$(PATH)
SHELL=/bin/bash

.PHONY: start
start:

	chastifol [ make watchify ] [ make sassify ] [ make server ]

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

.PHONY: watchify
watchify:

	mkdir -p build
	watchify -d -v -t babelify ./src/client/index.js -o ./build/index.js

.PHONY: sass
sass:

	mkdir -p build
	node-sass ./src/scss/style.scss -o ./build/

.PHONY: sassify
sassify: sass

	mkdir -p build
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

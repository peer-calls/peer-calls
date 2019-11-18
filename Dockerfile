FROM node:12-alpine
WORKDIR /app
RUN chown node:node /app
USER node
COPY . .
RUN npm install
RUN npm run build
RUN rm -rf node_modules

FROM node:12-alpine
WORKDIR /app
RUN chown node:node /app
COPY --from=0 /app .
RUN npm install --production
USER root
RUN chown -R root:root .
USER node
EXPOSE 3000
CMD ["node", "lib/index.js"]

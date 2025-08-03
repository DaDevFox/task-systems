FROM node:20 as build
WORKDIR /app
COPY package.json .
RUN npm install
RUN npm install -g @protobuf-ts/plugin
COPY . .
RUN  protoc   --proto_path=./proto   --js_out=import_style=commonjs:./frontend/src/proto   --grpc-web_out=import_style=typescript,mode=grpcwebtext:./frontend/src/proto   proto/*.proto
# RUN protoc --proto_path=proto/ --plugin=/usr/local/bin/protoc-gen-ts=frontend/node_modules/.bin/protoc-gen-ts   --js_out=import_style=commonjs,binary:frontend/src/proto   --ts_out=service=grpc-web:./frontend/src/proto   -I . cooking.proto tasks.proto state.proto config.proto hometasker_service.proto
RUN npm run build

FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html


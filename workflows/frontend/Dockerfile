# Build stage
FROM node:20-alpine AS build
WORKDIR /app
COPY package.json .
RUN npm install
RUN npm install -g @protobuf-ts/plugin
COPY . .
RUN  protoc \
  --proto_path=./proto \
  --js_out=import_style=commonjs:./frontend/src/proto \
  --grpc-web_out=import_style=typescript,mode=grpcwebtext:./frontend/src/proto \
  proto/*.proto
RUN npm run build

# Runtime stage
FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html

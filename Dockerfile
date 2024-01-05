# Étape de construction
FROM golang:1.21 AS build

# Définir le répertoire de travail
WORKDIR /app

# Copier les fichiers go mod et sum
COPY go.mod go.sum ./

# Télécharger toutes les dépendances
RUN go mod download

# Copier le code source dans le conteneur Docker
COPY . .

# Compiler l'application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Étape finale
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copier le binaire de l'étape de construction
COPY --from=build /app/main .

# Exposer le port sur lequel votre application s'exécute
EXPOSE 8080

# Exécuter le binaire
CMD ["./main"]
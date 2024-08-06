### Observações a respeito do projeto GO

Geralmente iniciamos o módulo nomeando-o com o nome do repositório

    go mod init https://github.com/guilherme25alves/{nome-repositorio}

#### Lembrete no momento de se conectar ao banco de dados

O banco deve ser referenciado no host pelo nome do serviço no Docker, no caso, foi nomeado como `db`.

### Ferramenta utilizada para migrations 

    jackc -> Tern -> lib do GO

    go install github.com/jackc/tern/v2@latest

Comando para inicializar o TERN, vai criar um arquivo de config e outro de exemplo de uma migration

    `tern init ./internal/store/pgstore/migrations`

Deve ser preenchido as propriedades de [database] no tern.conf 

Para criar uma nova migration: 

    `tern new --migrations ./internal/store/pgstore/migrations create_rooms_table` --> passando o caminho onde queremos salvar e o nome da migration

Foi criado um arquivo em GO para fazer Wrap do tern, pois ele não aceita por padrão carregar variáveis de ambiente
com arquivos .env. Para executar, precisamos rodar o módulo

    `go run cmd/tools/terndotenv/main.go`

### Lib para gerar códigos SQL em GO 

- sqlc (não é um ORM -> em GO não é recomendado ORM)

    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

Necessário criar um arquivo de config, geralmente `sqlc.yml`, na pasta de migrations
Após configuração feita, rodar com o comando 

    sqlc generate -f ./internal/store/pgstore/sqlc.yml

### CHI -> Lib para rest API com GO 

Está sendo utilizado a lib CHI. 

Um dos problemas comuns ao implementar uma API é o caso dos CORs que podem bloquear os acessos as requisições. Felizmente, o CHI possui um pacote que lida com cors. 

Para fazer o download dessa lib, devemos executar o comando: 

    `go get github.com/go-chi/cors`


### Gorilla Websocket

Pacote do GO que está sendo utilizado para criar um Websocket --> gorrila/websocket

Para fazer o download, executar o comando: 

    `go get github.com/gorilla/websocket`
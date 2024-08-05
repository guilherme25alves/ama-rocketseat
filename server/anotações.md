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


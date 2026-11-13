package i18n

var operateMessages = map[string][3]string{
	// --- logs ---
	"cmd.logs.short": {
		"Show the logs of the app, a service or the database",
		"Muestra los logs de la app, de un servicio o de la base",
		"Mostra os logs do app, de um serviço ou do banco",
	},
	"flag.log_service": {"logs of this service", "logs de este servicio", "logs deste serviço"},
	"flag.log_container": {
		"app (default), db or storage",
		"app (por defecto), db o storage",
		"app (padrão), db ou storage",
	},
	"flag.tail":   {"number of lines, up to 2000", "cantidad de líneas, hasta 2000", "quantidade de linhas, até 2000"},
	"flag.follow": {"keep showing new lines", "sigue mostrando las líneas nuevas", "continua mostrando as linhas novas"},
	"logs.gap": {
		"(more than %d lines arrived in 2 s: some were skipped)",
		"(llegaron más de %d líneas en 2 s: se saltaron algunas)",
		"(chegaram mais de %d linhas em 2 s: algumas foram puladas)",
	},
	"logs.bad_container": {
		"unknown container %q: use app, db or storage (or --service <name>)",
		"contenedor desconocido %q: usa app, db o storage (o --service <nombre>)",
		"contêiner desconhecido %q: use app, db ou storage (ou --service <nome>)",
	},

	// --- config / restart ---
	"cmd.config.short": {
		"Project settings (auth, SMTP, CORS…)",
		"Configuración del proyecto (auth, SMTP, CORS…)",
		"Configuração do projeto (auth, SMTP, CORS…)",
	},
	"cmd.config.get.short": {"Show the settings", "Muestra la configuración", "Mostra a configuração"},
	"cmd.config.set.short": {"Change settings", "Cambia la configuración", "Muda a configuração"},
	"config.bad_pair": {
		"%q is not KEY=VALUE",
		"%q no es CLAVE=VALOR",
		"%q não é CHAVE=VALOR",
	},
	"config.saved": {
		"Saved %s (the instance restarts to apply them)",
		"Guardado %s (la instancia se reinicia para aplicarlo)",
		"Salvo %s (a instância reinicia para aplicar)",
	},
	"config.rejected": {
		"Not accepted (outside the allowed variables): %s",
		"No aceptadas (fuera de las variables permitidas): %s",
		"Não aceitas (fora das variáveis permitidas): %s",
	},
	"cmd.restart.short": {"Restart the instance", "Reinicia la instancia", "Reinicia a instância"},
	"cmd.tables.short":  {"Tables and their columns", "Tablas y sus columnas", "Tabelas e suas colunas"},
	"cmd.tables.long": {
		"Lists the tables (rows, primary key and columns) or describes one: types, nulls, defaults and foreign keys. It only reads, so it works with a read-only session (unlike `sql`).",
		"Lista las tablas (filas, clave primaria y columnas) o describe una: tipos, nulos, valores por defecto y claves foráneas. Solo lee, así que funciona con una sesión de solo lectura (a diferencia de `sql`).",
		"Lista as tabelas (linhas, chave primária e colunas) ou descreve uma: tipos, nulos, valores padrão e chaves estrangeiras. Só lê, então funciona com uma sessão somente leitura (ao contrário do `sql`).",
	},
	"tables.none":  {"No tables yet.", "Todavía no hay tablas.", "Ainda não há tabelas."},
	"tables.rows":  {"%d rows", "%d filas", "%d linhas"},
	"restart.done": {"Instance restarted", "Instancia reiniciada", "Instância reiniciada"},

	// --- sql ---
	"cmd.sql.short": {"Run SQL on the database", "Corre SQL en la base", "Roda SQL no banco"},
	"cmd.sql.long": {
		"Run SQL on the project's Postgres: one query with $1, $2… parameters, or a script of several statements separated by ; (then without parameters and without rows). It needs write access; with --read it runs one statement as a user that can only read, inside a read-only transaction, and a read-only connection is enough.",
		"Corre SQL en el Postgres del proyecto: una consulta con parámetros $1, $2…, o un script de varias sentencias separadas por ; (entonces sin parámetros y sin filas). Pide escritura; con --read corre una sola sentencia como un usuario que solo puede leer, en una transacción de solo lectura, y alcanza con una conexión de lectura.",
		"Roda SQL no Postgres do projeto: uma consulta com parâmetros $1, $2…, ou um script de várias instruções separadas por ; (aí sem parâmetros e sem linhas). Exige escrita; com --read roda uma só instrução como um usuário que só pode ler, numa transação somente leitura, e basta uma conexão de leitura.",
	},
	"flag.sql_file": {"read the SQL from a file", "lee el SQL de un archivo", "lê o SQL de um arquivo"},
	"flag.sql_read": {
		"read only: one statement that cannot write (works with a read-only connection)",
		"solo lectura: una sentencia que no puede escribir (sirve con una conexión de lectura)",
		"somente leitura: uma instrução que não pode escrever (serve com uma conexão de leitura)",
	},
	"sql.read_hint": {
		"to only read, add --read",
		"para solo leer, agrega --read",
		"para só ler, adicione --read",
	},
	"flag.sql_param": {
		"value for $1, $2… in order (numbers, true, false and null go as such)",
		"valor para $1, $2… en orden (números, true, false y null van como tales)",
		"valor para $1, $2… em ordem (números, true, false e null vão como tais)",
	},
	"flag.sql_params": {
		`parameters as a JSON array, e.g. '["123", 4]'`,
		`parámetros como array JSON, p. ej. '["123", 4]'`,
		`parâmetros como array JSON, p. ex. '["123", 4]'`,
	},
	"flag.sql_timeout": {
		"seconds before Postgres cancels it (default 60, max 600)",
		"segundos antes de que Postgres la cancele (60 por defecto, hasta 600)",
		"segundos antes de o Postgres cancelar (60 por padrão, até 600)",
	},
	"sql.params_both": {
		"use --param or --params, not both",
		"usa --param o --params, no los dos",
		"use --param ou --params, não os dois",
	},
	"sql.bad_params": {
		"--params must be a JSON array",
		"--params tiene que ser un array JSON",
		"--params precisa ser um array JSON",
	},
	"sql.running":  {"Running…", "Corriendo…", "Rodando…"},
	"sql.rows":     {"%d row(s) · %s", "%d fila(s) · %s", "%d linha(s) · %s"},
	"sql.affected": {"%d row(s) affected · %s", "%d fila(s) afectada(s) · %s", "%d linha(s) afetada(s) · %s"},
	"sql.query_and_file": {
		"pass the query or --file, not both",
		"pasa la consulta o --file, no los dos",
		"passe a consulta ou --file, não os dois",
	},
	"sql.reading_stdin": {
		"Reading the SQL from stdin (end with Ctrl+D, or Ctrl+Z and Enter on Windows)…",
		"Leyendo el SQL de stdin (termina con Ctrl+D, o Ctrl+Z y Enter en Windows)…",
		"Lendo o SQL do stdin (termine com Ctrl+D, ou Ctrl+Z e Enter no Windows)…",
	},
	"sql.no_file": {"could not read %s", "no se pudo leer %s", "não foi possível ler %s"},
	"sql.empty": {
		"there is no SQL: pass it as an argument, with --file, or with - to read stdin",
		"no hay SQL: pásalo como argumento, con --file, o con - para leer stdin",
		"não há SQL: passe como argumento, com --file, ou com - para ler o stdin",
	},

	// --- api ---
	"cmd.api.short": {
		"Call the project's API with the service key",
		"Llama a la API del proyecto con la service key",
		"Chama a API do projeto com a service key",
	},
	"cmd.api.long": {
		"Call the project's HTTP API (REST, storage, auth admin…) with the service key, which never leaves Rapibase. The path goes under /api/. GET needs read access; the rest, write.",
		"Llama a la API HTTP del proyecto (REST, storage, admin de auth…) con la service key, que nunca sale de Rapibase. El path va bajo /api/. GET pide lectura; el resto, escritura.",
		"Chama a API HTTP do projeto (REST, storage, admin de auth…) com a service key, que nunca sai do Rapibase. O caminho vai sob /api/. GET exige leitura; o resto, escrita.",
	},
	"flag.api_method": {
		"HTTP method (GET by default, POST with -d or -F)",
		"método HTTP (GET por defecto, POST con -d o -F)",
		"método HTTP (GET por padrão, POST com -d ou -F)",
	},
	"flag.api_data": {
		"body: text, @file or - (stdin)",
		"cuerpo: texto, @archivo o - (stdin)",
		"corpo: texto, @arquivo ou - (stdin)",
	},
	"flag.api_form": {
		"multipart field name=value or name=@file (repeatable)",
		"campo multipart nombre=valor o nombre=@archivo (se puede repetir)",
		"campo multipart nome=valor ou nome=@arquivo (pode repetir)",
	},
	"flag.api_content_type": {
		"Content-Type of -d (by default JSON if it is valid JSON)",
		"Content-Type de -d (por defecto JSON si es JSON válido)",
		"Content-Type do -d (por padrão JSON se for JSON válido)",
	},
	"flag.api_output": {
		"save the response to a file",
		"guarda la respuesta en un archivo",
		"salva a resposta em um arquivo",
	},
	"flag.api_anon": {
		"call as a visitor without a session (the project's anon key; it is never shown)",
		"llama como un visitante sin sesión (la anon key del proyecto; nunca se muestra)",
		"chama como um visitante sem sessão (a anon key do projeto; nunca é mostrada)",
	},
	"flag.api_jwt": {
		"call as the user of this JWT (anon key + Authorization: Bearer <jwt>)",
		"llama como el usuario de ese JWT (anon key + Authorization: Bearer <jwt>)",
		"chama como o usuário desse JWT (anon key + Authorization: Bearer <jwt>)",
	},
	"api.anon_and_jwt": {
		"use --anon or --jwt, not both",
		"usa --anon o --jwt, no los dos",
		"use --anon ou --jwt, não os dois",
	},
	"api.data_and_form": {
		"use -d or -F, not both",
		"usa -d o -F, no los dos",
		"use -d ou -F, não os dois",
	},
	"api.bad_method": {
		"unsupported method %q: GET, POST, PUT, PATCH or DELETE",
		"método no soportado %q: GET, POST, PUT, PATCH o DELETE",
		"método não suportado %q: GET, POST, PUT, PATCH ou DELETE",
	},
	"api.two_methods": {
		"two methods: %s and -X %s",
		"dos métodos: %s y -X %s",
		"dois métodos: %s e -X %s",
	},
	"api.status": {"the API answered %d %s", "la API respondió %d %s", "a API respondeu %d %s"},
	"api.saved":  {"Saved %s (%d bytes)", "Guardado %s (%d bytes)", "Salvo %s (%d bytes)"},
	"api.binary": {
		"the response is binary (%d bytes): save it with -o",
		"la respuesta es binaria (%d bytes): guárdala con -o",
		"a resposta é binária (%d bytes): salve com -o",
	},
	"api.bad_path": {
		"invalid path %q: it goes under /api/ and without ..",
		"path inválido %q: va bajo /api/ y sin ..",
		"caminho inválido %q: vai sob /api/ e sem ..",
	},
	"api.no_file": {"could not read %s", "no se pudo leer %s", "não foi possível ler %s"},
	"api.bad_form": {
		"%q is not name=value or name=@file",
		"%q no es nombre=valor ni nombre=@archivo",
		"%q não é nome=valor nem nome=@arquivo",
	},

	// --- rls ---
	"cmd.rls.short": {
		"Row-level security of the tables",
		"Seguridad por filas de las tablas",
		"Segurança por linhas das tabelas",
	},
	"cmd.rls.long": {
		"See and change the row-level security of the tables, with the same modes as the dashboard. Without a subcommand it lists the protected tables.",
		"Mira y cambia la seguridad por filas de las tablas, con los mismos modos que el dashboard. Sin subcomando lista las tablas protegidas.",
		"Veja e mude a segurança por linhas das tabelas, com os mesmos modos do dashboard. Sem subcomando lista as tabelas protegidas.",
	},
	"cmd.rls.list.short": {
		"List the protected tables, or the detail of one",
		"Lista las tablas protegidas, o el detalle de una",
		"Lista as tabelas protegidas, ou o detalhe de uma",
	},
	"cmd.rls.set.short": {"Protect a table", "Protege una tabla", "Protege uma tabela"},
	"cmd.rls.set.long": {
		"Protect a table. Modes: owner (each user sees only their rows; needs --owner-column, a uuid column), authenticated (any logged-in user), public (anyone reads, logged-in users write), read_only (anyone reads, nobody writes through the API) and custom (no access until you add your own policies with `rapibase sql`). The service key always bypasses RLS.",
		"Protege una tabla. Modos: owner (cada usuario ve solo sus filas; necesita --owner-column, una columna uuid), authenticated (cualquier usuario con sesión), public (cualquiera lee, los usuarios con sesión escriben), read_only (cualquiera lee, nadie escribe por la API) y custom (sin acceso hasta que agregues tus políticas con `rapibase sql`). La service key siempre saltea el RLS.",
		"Protege uma tabela. Modos: owner (cada usuário vê só suas linhas; precisa de --owner-column, uma coluna uuid), authenticated (qualquer usuário logado), public (qualquer um lê, usuários logados escrevem), read_only (qualquer um lê, ninguém escreve pela API) e custom (sem acesso até você adicionar suas políticas com `rapibase sql`). A service key sempre ignora o RLS.",
	},
	"cmd.rls.disable.short": {
		"Remove the protection of a table",
		"Quita la protección de una tabla",
		"Remove a proteção de uma tabela",
	},
	"flag.owner_column": {
		"uuid column with the owner's id (mode owner)",
		"columna uuid con el id del dueño (modo owner)",
		"coluna uuid com o id do dono (modo owner)",
	},
	"rls.none": {"No protected tables.", "No hay tablas protegidas.", "Não há tabelas protegidas."},
	"rls.bad_mode": {
		"unknown mode %q: %s",
		"modo desconocido %q: %s",
		"modo desconhecido %q: %s",
	},
	"rls.owner_needs_column": {
		"the owner mode needs the column with the owner's id",
		"el modo owner necesita la columna con el id del dueño",
		"o modo owner precisa da coluna com o id do dono",
	},
	"rls.set": {"%s is protected (%s)", "%s quedó protegida (%s)", "%s está protegida (%s)"},
	"rls.custom_note": {
		"Nobody reaches it through the API until you add policies: rapibase sql \"create policy …\"",
		"Nadie la alcanza por la API hasta que agregues políticas: rapibase sql \"create policy …\"",
		"Ninguém a alcança pela API até você adicionar políticas: rapibase sql \"create policy …\"",
	},
	"rls.confirm_disable": {
		"Remove the protection of %s? Any key will reach every row",
		"¿Quitar la protección de %s? Cualquier clave alcanzará todas las filas",
		"Remover a proteção de %s? Qualquer chave alcançará todas as linhas",
	},
	"rls.disable_needs_yes": {
		"removing the protection of %s opens every row: confirm with --yes",
		"quitar la protección de %s abre todas las filas: confirma con --yes",
		"remover a proteção de %s abre todas as linhas: confirme com --yes",
	},
	"rls.disabled": {"%s is no longer protected", "%s ya no está protegida", "%s não está mais protegida"},
}

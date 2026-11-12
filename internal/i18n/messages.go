package i18n

// messages: key → {en, es, pt}. Spanish uses "tú", like the console.
// They are split by area in messages_*.go and merged here.
var messages = map[string][3]string{}

func add(m map[string][3]string) {
	for k, v := range m {
		if _, dup := messages[k]; dup {
			panic("i18n: duplicate key " + k)
		}
		messages[k] = v
	}
}

func init() {
	add(baseMessages)
	add(projectMessages)
	add(publishMessages)
	add(operateMessages)
	add(agentMessages)
}

var baseMessages = map[string][3]string{
	"tagline": {
		"the backend for apps built with coding agents",
		"el backend para apps hechas con agentes de código",
		"o backend para apps feitos com agentes de código",
	},

	// --- help ---
	"help.usage":    {"Usage", "Uso", "Uso"},
	"help.command":  {"command", "comando", "comando"},
	"help.options":  {"options", "opciones", "opções"},
	"help.commands": {"Commands", "Comandos", "Comandos"},
	"help.flags":    {"Options", "Opciones", "Opções"},
	"help.global":   {"Global options", "Opciones globales", "Opções globais"},
	"help.examples": {"Examples", "Ejemplos", "Exemplos"},
	"help.more":     {"More at", "Más en", "Mais em"},

	"help.group.start":   {"Start", "Empezar", "Começar"},
	"help.group.project": {"Project", "Proyecto", "Projeto"},
	"help.group.publish": {"Publish", "Publicar", "Publicar"},
	"help.group.operate": {"Operate", "Operar", "Operar"},
	"help.group.agent":   {"Your agent", "Tu agente", "Seu agente"},

	// --- global flags ---
	"flag.project": {
		"project to act on (slug or id); by default the linked one",
		"proyecto sobre el que actuar (slug o id); por defecto, el vinculado",
		"projeto em que agir (slug ou id); por padrão, o vinculado",
	},
	"flag.json": {
		"print one JSON object on stdout (for agents and scripts)",
		"imprime un único objeto JSON en stdout (para agentes y scripts)",
		"imprime um único objeto JSON no stdout (para agentes e scripts)",
	},
	"flag.yes": {
		"accept the defaults and confirm without asking",
		"acepta los valores por defecto y confirma sin preguntar",
		"aceita os valores padrão e confirma sem perguntar",
	},
	"flag.lang":       {"language of the messages", "idioma de los mensajes", "idioma das mensagens"},
	"flag.no_browser": {"do not open the browser", "no abre el navegador", "não abre o navegador"},
	"flag.debug": {
		"show the HTTP requests on stderr (never keys or bodies)",
		"muestra los pedidos HTTP en stderr (nunca claves ni cuerpos)",
		"mostra as requisições HTTP no stderr (nunca chaves nem corpos)",
	},
	"flag.version":   {"print the version", "imprime la versión", "imprime a versão"},
	"flag.help":      {"help for this command", "ayuda de este comando", "ajuda deste comando"},
	"flag.help_root": {"help for any command", "ayuda de cualquier comando", "ajuda de qualquer comando"},

	// --- generic errors ---
	"err.bad_lang": {
		"unsupported language %q: use es, en or pt",
		"idioma %q no soportado: usa es, en o pt",
		"idioma %q não suportado: use es, en ou pt",
	},
	"err.available_projects": {"Your projects:", "Tus proyectos:", "Seus projetos:"},
	"err.missing_flags":      {"Missing: %s", "Falta: %s", "Falta: %s"},
	"err.retry_after":        {"Try again in %s s.", "Prueba de nuevo en %s s.", "Tente de novo em %s s."},
	"err.unknown_command":    {"unknown command %q", "comando desconocido %q", "comando desconhecido %q"},
	"err.did_you_mean":       {"Did you mean `rapibase %s`?", "¿Quisiste decir `rapibase %s`?", "Você quis dizer `rapibase %s`?"},
	"err.see_help":           {"See `rapibase --help`.", "Mira `rapibase --help`.", "Veja `rapibase --help`."},

	// --- api client ---
	"api.bad_response": {
		"unexpected response from Rapibase (HTTP %d)",
		"respuesta inesperada de Rapibase (HTTP %d)",
		"resposta inesperada do Rapibase (HTTP %d)",
	},
	"api.network": {
		"could not reach Rapibase: %s",
		"no se pudo conectar con Rapibase: %s",
		"não foi possível conectar ao Rapibase: %s",
	},
	"api.http_status": {"Rapibase answered HTTP %d", "Rapibase respondió HTTP %d", "O Rapibase respondeu HTTP %d"},

	// --- ui ---
	"ui.choose":         {"Choose 1-%d", "Elige 1-%d", "Escolha 1-%d"},
	"ui.invalid_choice": {"Not a valid option.", "No es una opción válida.", "Não é uma opção válida."},

	// --- auth ---
	"auth.needs_login": {
		"you are not logged in: run `rapibase login`",
		"no iniciaste sesión: corre `rapibase login`",
		"você não está logado: rode `rapibase login`",
	},
	"auth.env_token_rejected": {
		"RAPIBASE_TOKEN was rejected: it expired or was revoked",
		"RAPIBASE_TOKEN fue rechazado: venció o lo revocaron",
		"RAPIBASE_TOKEN foi rejeitado: expirou ou foi revogado",
	},
	"auth.deploy_token_rejected": {
		"RAPIBASE_DEPLOY_TOKEN was rejected or does not cover this action",
		"RAPIBASE_DEPLOY_TOKEN fue rechazado o no cubre esta acción",
		"RAPIBASE_DEPLOY_TOKEN foi rejeitado ou não cobre esta ação",
	},
	"auth.session_expired": {
		"your session expired: run `rapibase login`",
		"tu sesión venció: corre `rapibase login`",
		"sua sessão expirou: rode `rapibase login`",
	},
	"auth.register_rate_limited": {
		"too many sign-ins from this network; try again in a few minutes",
		"demasiados inicios de sesión desde esta red; prueba en unos minutos",
		"logins demais a partir desta rede; tente em alguns minutos",
	},
	"auth.register_failed": {
		"could not register the CLI: %s",
		"no se pudo registrar el CLI: %s",
		"não foi possível registrar o CLI: %s",
	},
	"auth.device_rate_limited": {
		"too many login codes requested; try again in a few minutes",
		"se pidieron demasiados códigos; prueba en unos minutos",
		"códigos demais solicitados; tente em alguns minutos",
	},
	"auth.client_reset": {
		"this CLI's registration was removed; run the command again to register it",
		"el registro de este CLI se borró; corre el comando de nuevo para registrarlo",
		"o registro deste CLI foi removido; rode o comando de novo para registrá-lo",
	},
	"auth.device_failed": {
		"could not start the login: %s",
		"no se pudo iniciar el login: %s",
		"não foi possível iniciar o login: %s",
	},
	"auth.deploy_only": {
		"RAPIBASE_DEPLOY_TOKEN only works for deploys; run `rapibase login` for the rest",
		"RAPIBASE_DEPLOY_TOKEN solo sirve para deploys; corre `rapibase login` para lo demás",
		"RAPIBASE_DEPLOY_TOKEN só serve para deploys; rode `rapibase login` para o resto",
	},

	// --- login / logout / whoami ---
	"cmd.login.short": {"Log in with the browser", "Inicia sesión con el navegador", "Faz login pelo navegador"},
	"cmd.login.long": {
		"Log in with the browser (device flow). Without a terminal it prints the link and the code, and running it again waits for the approval.",
		"Inicia sesión con el navegador (device flow). Sin terminal imprime el enlace y el código, y al correrlo de nuevo espera la aprobación.",
		"Faz login pelo navegador (device flow). Sem terminal imprime o link e o código, e ao rodar de novo espera a aprovação.",
	},
	"flag.wait_login": {
		"wait for the approval (up to 90 s without a terminal)",
		"espera la aprobación (hasta 90 s sin terminal)",
		"espera a aprovação (até 90 s sem terminal)",
	},
	"flag.ref": {"partner code", "código de partner", "código de parceiro"},
	"login.env_token": {
		"RAPIBASE_TOKEN is set: the CLI uses it and does not need a login",
		"RAPIBASE_TOKEN está definida: el CLI la usa y no hace falta login",
		"RAPIBASE_TOKEN está definida: o CLI a usa e não precisa de login",
	},
	"login.already": {"Already logged in as %s", "Ya tienes sesión como %s", "Você já está logado como %s"},
	"login.pending": {
		"login pending: open the link, approve, and run the command again",
		"login pendiente: abre el enlace, aprueba y corre el comando de nuevo",
		"login pendente: abra o link, aprove e rode o comando de novo",
	},
	"login.opened":    {"We opened your browser. If it didn't open:", "Abrimos tu navegador. Si no se abrió:", "Abrimos seu navegador. Se não abriu:"},
	"login.open_link": {"Open this link and approve:", "Abre este enlace y aprueba:", "Abra este link e aprove:"},
	"login.code":      {"code %s", "código %s", "código %s"},
	"login.waiting":   {"Waiting for the approval…", "Esperando la aprobación…", "Esperando a aprovação…"},
	"login.denied":    {"the login was denied in the browser", "el login se rechazó en el navegador", "o login foi recusado no navegador"},
	"login.expired":   {"the login code expired", "el código de login venció", "o código de login expirou"},
	"login.expired_hint": {
		"Run `rapibase login` for a new one.",
		"Corre `rapibase login` para pedir otro.",
		"Rode `rapibase login` para pedir outro.",
	},
	"login.done": {"Logged in as %s", "Sesión iniciada como %s", "Login feito como %s"},

	"cmd.logout.short": {"Log out on this machine", "Cierra la sesión en esta máquina", "Encerra a sessão nesta máquina"},
	"logout.env_token": {
		"the session comes from a variable (%s): unset it to log out",
		"la sesión viene de una variable (%s): quítala para cerrar sesión",
		"a sessão vem de uma variável (%s): remova-a para sair",
	},
	"logout.none": {"There was no session", "No había sesión", "Não havia sessão"},
	"logout.done": {"Logged out", "Sesión cerrada", "Sessão encerrada"},
	"logout.local_only": {
		"Logged out here (the server could not be reached to revoke the token)",
		"Sesión cerrada aquí (no se pudo avisar al servidor para revocar el token)",
		"Sessão encerrada aqui (não deu para avisar o servidor para revogar o token)",
	},

	"cmd.whoami.short": {
		"Show the account and what the session reaches",
		"Muestra la cuenta y lo que alcanza la sesión",
		"Mostra a conta e o que a sessão alcança",
	},
	"whoami.account":       {"Account:", "Cuenta:", "Conta:"},
	"whoami.session":       {"Session:", "Sesión:", "Sessão:"},
	"whoami.reach":         {"Reaches:", "Alcanza:", "Alcança:"},
	"whoami.some_projects": {"%d project(s)", "%d proyecto(s)", "%d projeto(s)"},
	"whoami.all_projects":  {"all your projects (%s)", "todos tus proyectos (%s)", "todos os seus projetos (%s)"},
	"whoami.can_create": {
		"This session can create free projects.",
		"Esta sesión puede crear proyectos gratis.",
		"Esta sessão pode criar projetos grátis.",
	},
	"whoami.manage":    {"Change what it reaches: %s", "Cambia lo que alcanza: %s", "Mude o que ela alcança: %s"},
	"whoami.src_login": {"rapibase login", "rapibase login", "rapibase login"},
	"mode.write":       {"read and write", "lectura y escritura", "leitura e escrita"},
	"mode.read":        {"read only", "solo lectura", "somente leitura"},

	"version.unsupported": {
		"this CLI (%s) is too old for Rapibase: %s or newer is needed",
		"este CLI (%s) es demasiado viejo para Rapibase: hace falta %s o más nuevo",
		"este CLI (%s) é antigo demais para o Rapibase: precisa da %s ou mais nova",
	},
	"version.newer": {
		"rapibase %s is out (you have %s): npm i -g rapibase@latest",
		"salió rapibase %s (tienes %s): npm i -g rapibase@latest",
		"saiu o rapibase %s (você tem %s): npm i -g rapibase@latest",
	},
}

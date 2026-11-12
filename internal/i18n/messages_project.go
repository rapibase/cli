package i18n

var projectMessages = map[string][3]string{
	// --- projects / create ---
	"cmd.projects.short": {
		"List your projects, grouped by organization",
		"Lista tus proyectos, agrupados por organización",
		"Lista seus projetos, agrupados por organização",
	},
	"projects.empty": {"You have no projects yet.", "Todavía no tienes proyectos.", "Você ainda não tem projetos."},
	"cmd.projects.create.short": {
		"Create a free project",
		"Crea un proyecto gratis",
		"Cria um projeto grátis",
	},
	"cmd.projects.create.long": {
		"Create a free project (or one covered by your Team). Paid projects are created in the console.",
		"Crea un proyecto gratis (o uno cubierto por tu Team). Los proyectos pagos se crean en la consola.",
		"Cria um projeto grátis (ou um coberto pelo seu Team). Projetos pagos são criados no console.",
	},
	"flag.name":         {"project name", "nombre del proyecto", "nome do projeto"},
	"flag.org":          {"organization (name or id)", "organización (nombre o id)", "organização (nome ou id)"},
	"flag.region":       {"region of the new project", "región del proyecto nuevo", "região do projeto novo"},
	"flag.wait_running": {"wait until it is running", "espera hasta que esté corriendo", "espera até estar rodando"},
	"create.no_scope": {
		"this session cannot create projects: allow it in the console",
		"esta sesión no puede crear proyectos: habilítalo en la consola",
		"esta sessão não pode criar projetos: habilite no console",
	},
	"create.bad_name": {
		"use letters, numbers, spaces, dots, - or _ (up to 40)",
		"usa letras, números, espacios, puntos, - o _ (hasta 40)",
		"use letras, números, espaços, pontos, - ou _ (até 40)",
	},
	"create.need_name": {"the project needs a name", "el proyecto necesita un nombre", "o projeto precisa de um nome"},
	"create.ask_name":  {"Project name", "Nombre del proyecto", "Nome do projeto"},
	"create.need_region": {
		"there are several regions: pass --region (%s)",
		"hay varias regiones: pasa --region (%s)",
		"há várias regiões: passe --region (%s)",
	},
	"create.ask_region": {"Region", "Región", "Região"},
	"create.creating":   {"Creating the project…", "Creando el proyecto…", "Criando o projeto…"},
	"create.free_used_hint": {
		"Your free project is %s: use it with `rapibase link`, or create a paid one in the console.",
		"Tu proyecto gratis es %s: úsalo con `rapibase link`, o crea uno pago en la consola.",
		"Seu projeto grátis é %s: use-o com `rapibase link`, ou crie um pago no console.",
	},
	"create.done": {"Project %s created", "Proyecto %s creado", "Projeto %s criado"},
	"create.bad_org": {
		"no organization %q (yours: %s)",
		"no hay una organización %q (las tuyas: %s)",
		"não há organização %q (as suas: %s)",
	},
	"create.ask_org": {"Organization", "Organización", "Organização"},
	"org.personal":   {"personal", "personal", "pessoal"},
	"plan.free":      {"Free", "Gratis", "Grátis"},

	// --- status of a project ---
	"status.waiting": {
		"Waiting for %s to start…",
		"Esperando a que arranque %s…",
		"Esperando %s iniciar…",
	},
	"status.running_after": {
		"Running at %s (%d s)",
		"Corriendo en %s (%d s)",
		"Rodando em %s (%d s)",
	},
	"status.failed_err": {
		"the project failed to start: %s",
		"el proyecto no pudo arrancar: %s",
		"o projeto não conseguiu iniciar: %s",
	},
	"status.paused_err": {
		"the project is paused: resume it in the console",
		"el proyecto está pausado: reanúdalo en la consola",
		"o projeto está pausado: retome-o no console",
	},
	"status.still_provisioning": {
		"the project is still starting",
		"el proyecto todavía está arrancando",
		"o projeto ainda está iniciando",
	},
	"status.running":      {"running", "corriendo", "rodando"},
	"status.provisioning": {"starting", "arrancando", "iniciando"},
	"status.paused":       {"paused", "pausado", "pausado"},
	"status.error":        {"error", "error", "erro"},
	"status.deleting":     {"deleting", "borrándose", "sendo apagado"},

	"cmd.status.short": {
		"Show the project: URLs, plan, services and usage",
		"Muestra el proyecto: URLs, plan, servicios y uso",
		"Mostra o projeto: URLs, plano, serviços e uso",
	},
	"status.plan":     {"Plan", "Plan", "Plano"},
	"status.console":  {"Console", "Consola", "Console"},
	"status.domain":   {"Domain", "Dominio", "Domínio"},
	"status.release":  {"Release", "Release", "Release"},
	"status.service":  {"Service", "Servicio", "Serviço"},
	"status.mode":     {"Access", "Acceso", "Acesso"},
	"status.src_link": {"linked in %s", "vinculado en %s", "vinculado em %s"},
	"status.src_flag": {"from --project", "por --project", "por --project"},
	"status.src_env":  {"from RAPIBASE_PROJECT", "por RAPIBASE_PROJECT", "por RAPIBASE_PROJECT"},
	"status.src_only": {"your only project", "tu único proyecto", "seu único projeto"},
	"services.asleep": {"(asleep)", "(dormido)", "(dormindo)"},

	"free.db_bytes":      {"Database", "Base", "Banco"},
	"free.storage_bytes": {"Storage", "Storage", "Storage"},
	"free.egress_bytes":  {"Egress", "Tráfico", "Tráfego"},
	"free.pauses_at":     {"Pauses on", "Se pausa el", "Pausa em"},

	// --- link / unlink ---
	"cmd.link.short": {
		"Link this folder to a project",
		"Vincula esta carpeta a un proyecto",
		"Vincula esta pasta a um projeto",
	},
	"cmd.link.long": {
		"Link this folder to a project: writes .rapibase/project.json, which has no secrets and can be committed. Commands run in this folder or below act on it; in a monorepo each app can have its own.",
		"Vincula esta carpeta a un proyecto: escribe .rapibase/project.json, que no tiene secretos y se puede commitear. Los comandos que corras en esta carpeta o más abajo actúan sobre él; en un monorepo cada app puede tener el suyo.",
		"Vincula esta pasta a um projeto: escreve .rapibase/project.json, que não tem segredos e pode ir no commit. Os comandos rodados nesta pasta ou abaixo agem sobre ele; num monorepo cada app pode ter o seu.",
	},
	"flag.force_relink": {
		"replace the existing link without asking",
		"reemplaza el vínculo existente sin preguntar",
		"substitui o vínculo existente sem perguntar",
	},
	"link.pick":         {"Which project?", "¿Qué proyecto?", "Qual projeto?"},
	"link.need_project": {"say which project to link", "di qué proyecto vincular", "diga qual projeto vincular"},
	"link.already":      {"Already linked to %s", "Ya está vinculado a %s", "Já está vinculado a %s"},
	"link.relink_needs_force": {
		"%s already links %s: to link %s instead, pass --force",
		"%s ya vincula %s: para vincular %s en su lugar, pasa --force",
		"%s já vincula %s: para vincular %s no lugar, passe --force",
	},
	"link.relink_hint": {
		"Or act on another project once with --project.",
		"O actúa sobre otro proyecto una sola vez con --project.",
		"Ou aja em outro projeto uma única vez com --project.",
	},
	"link.confirm_relink": {
		"%s links %s. Link %s instead?",
		"%s vincula %s. ¿Vincular %s en su lugar?",
		"%s vincula %s. Vincular %s no lugar?",
	},
	"link.kept": {"Kept the link to %s", "Se mantiene el vínculo con %s", "Mantido o vínculo com %s"},
	"link.done": {"Linked to %s (%s)", "Vinculado a %s (%s)", "Vinculado a %s (%s)"},
	"cmd.unlink.short": {
		"Remove the link of this folder",
		"Quita el vínculo de esta carpeta",
		"Remove o vínculo desta pasta",
	},
	"unlink.none": {"This folder was not linked", "Esta carpeta no estaba vinculada", "Esta pasta não estava vinculada"},
	"unlink.done": {"Unlinked %s (%s removed)", "Desvinculado %s (se borró %s)", "Desvinculado %s (%s removido)"},

	// --- open ---
	"cmd.open.short": {
		"Open the console, the dashboard or the app",
		"Abre la consola, el dashboard o la app",
		"Abre o console, o dashboard ou o app",
	},
	"open.not_running": {
		"the project is %s: there is nothing to open yet",
		"el proyecto está %s: todavía no hay nada que abrir",
		"o projeto está %s: ainda não há nada para abrir",
	},
	"open.bad_target": {
		"%q is not something to open: console, dashboard, app or connections",
		"%q no es algo para abrir: console, dashboard, app o connections",
		"%q não é algo para abrir: console, dashboard, app ou connections",
	},
	"open.opened": {"Opened %s", "Abierto %s", "Aberto %s"},

	// --- which project (resolve) ---
	"project.not_found": {
		"no project %q among the ones this session reaches",
		"no hay un proyecto %q entre los que alcanza esta sesión",
		"não há projeto %q entre os que esta sessão alcança",
	},
	"project.see_list": {"See them with `rapibase projects`.", "Míralos con `rapibase projects`.", "Veja-os com `rapibase projects`."},
	"project.deploy_token_id": {
		"with RAPIBASE_DEPLOY_TOKEN the project goes by numeric id (--project or RAPIBASE_PROJECT)",
		"con RAPIBASE_DEPLOY_TOKEN el proyecto va por id numérico (--project o RAPIBASE_PROJECT)",
		"com RAPIBASE_DEPLOY_TOKEN o projeto vai por id numérico (--project ou RAPIBASE_PROJECT)",
	},
	"project.other_than_linked": {
		"acting on %s, not on the linked %s",
		"actuando sobre %s, no sobre el vinculado %s",
		"agindo em %s, não no vinculado %s",
	},
	"project.linked_gone": {
		"the linked project %s no longer exists or this session does not reach it",
		"el proyecto vinculado %s ya no existe o esta sesión no lo alcanza",
		"o projeto vinculado %s não existe mais ou esta sessão não o alcança",
	},
	"project.relink_hint": {
		"Link another one with `rapibase link`.",
		"Vincula otro con `rapibase link`.",
		"Vincule outro com `rapibase link`.",
	},
	"project.none": {
		"this session does not reach any project",
		"esta sesión no alcanza ningún proyecto",
		"esta sessão não alcança nenhum projeto",
	},
	"project.none_hint": {
		"Create one with `rapibase init` or `rapibase projects create`.",
		"Crea uno con `rapibase init` o `rapibase projects create`.",
		"Crie um com `rapibase init` ou `rapibase projects create`.",
	},
	"project.pick": {"Which project?", "¿Sobre qué proyecto?", "Em qual projeto?"},
	"project.offer_link": {
		"Link this folder to %s so you are not asked again?",
		"¿Vincular esta carpeta a %s para no preguntar de nuevo?",
		"Vincular esta pasta a %s para não perguntar de novo?",
	},
	"project.not_linked": {
		"this folder is not linked and you have several projects",
		"esta carpeta no está vinculada y tienes varios proyectos",
		"esta pasta não está vinculada e você tem vários projetos",
	},
	"project.not_linked_hint": {
		"Pass --project <slug>, or link the folder with `rapibase link <slug>`.",
		"Pasa --project <slug>, o vincula la carpeta con `rapibase link <slug>`.",
		"Passe --project <slug>, ou vincule a pasta com `rapibase link <slug>`.",
	},

	// --- env ---
	"cmd.env.short": {"Project variables", "Variables del proyecto", "Variáveis do projeto"},
	"cmd.env.pull.short": {
		"Write the URL and keys in .env",
		"Escribe la URL y las claves en .env",
		"Escreve a URL e as chaves no .env",
	},
	"cmd.env.pull.long": {
		"Write the project's URL and keys in .env (merged with what is there), add it to .gitignore and refuse if git tracks it. Values never reach the terminal.",
		"Escribe la URL y las claves del proyecto en .env (mezcladas con lo que ya hay), lo agrega al .gitignore y se niega si git lo trackea. Los valores nunca salen por la terminal.",
		"Escreve a URL e as chaves do projeto no .env (mescladas com o que já existe), adiciona ao .gitignore e recusa se o git o rastreia. Os valores nunca aparecem no terminal.",
	},
	"flag.framework": {
		"framework for the public prefixes (%s)",
		"framework para los prefijos públicos (%s)",
		"framework para os prefixos públicos (%s)",
	},
	"flag.env_file": {"file to write", "archivo a escribir", "arquivo a escrever"},
	"env.bad_framework": {
		"unknown framework %q: use %s",
		"framework desconocido %q: usa %s",
		"framework desconhecido %q: use %s",
	},
	"env.written": {"Wrote %s: %s", "Escrito %s: %s", "Escrito %s: %s"},
	"env.gitignored": {
		"Added %s to .gitignore",
		"Se agregó %s al .gitignore",
		"%s adicionado ao .gitignore",
	},
	"env.framework": {
		"%s: the public variables also go with the %s prefix",
		"%s: las variables públicas van también con el prefijo %s",
		"%s: as variáveis públicas vão também com o prefixo %s",
	},
	"env.tracked": {
		"git tracks %s: the keys would end up in the repo",
		"git trackea %s: las claves terminarían en el repo",
		"o git rastreia %s: as chaves acabariam no repo",
	},
}

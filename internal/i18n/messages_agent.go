package i18n

var agentMessages = map[string][3]string{
	// --- choosing the agent ---
	"flag.agent": {
		"claude, cursor, opencode, codex, hermes, other or none (repeatable)",
		"claude, cursor, opencode, codex, hermes, other o none (se puede repetir)",
		"claude, cursor, opencode, codex, hermes, other ou none (pode repetir)",
	},
	"flag.scope": {
		"project (the repo, shared), local (the repo, only you) or user (all your repos)",
		"project (el repo, compartido), local (el repo, solo tú) o user (todos tus repos)",
		"project (o repo, compartilhado), local (o repo, só você) ou user (todos os seus repos)",
	},
	"agent.bad_scope": {
		"unknown scope %q: project, local or user",
		"scope desconocido %q: project, local o user",
		"scope desconhecido %q: project, local ou user",
	},
	"agent.bad": {
		"unknown agent %q: %s",
		"agente desconocido %q: %s",
		"agente desconhecido %q: %s",
	},
	"agent.detected_running": {
		"Running inside %s",
		"Corriendo dentro de %s",
		"Rodando dentro de %s",
	},
	"agent.fallback_other": {
		"No agent detected: the skills go to .agents/skills with a line in AGENTS.md",
		"No se detectó el agente: los skills van a .agents/skills con una línea en AGENTS.md",
		"Nenhum agente detectado: os skills vão para .agents/skills com uma linha no AGENTS.md",
	},
	"agent.installed":  {"installed", "instalado", "instalado"},
	"agent.other_hint": {"any agent that reads AGENTS.md", "cualquier agente que lea AGENTS.md", "qualquer agente que leia AGENTS.md"},
	"agent.none":       {"None", "Ninguno", "Nenhum"},
	"agent.ask":        {"Which agent do you use?", "¿Qué agente usas?", "Qual agente você usa?"},

	// --- skill ---
	"cmd.skill.short": {
		"The Rapibase skills for your agent",
		"Los skills de Rapibase para tu agente",
		"Os skills do Rapibase para o seu agente",
	},
	"cmd.skill.install.short": {
		"Install the skills where your agent reads them",
		"Instala los skills donde los lee tu agente",
		"Instala os skills onde o seu agente os lê",
	},
	"cmd.skill.install.long": {
		"Install the four Rapibase skills (backend, deploy, functions, observe) where your agent reads them. They come with the CLI, so they always match its commands.",
		"Instala los cuatro skills de Rapibase (backend, deploy, functions, observe) donde los lee tu agente. Vienen con el CLI, así que siempre coinciden con sus comandos.",
		"Instala os quatro skills do Rapibase (backend, deploy, functions, observe) onde o seu agente os lê. Vêm com o CLI, então sempre batem com os comandos dele.",
	},
	"cmd.skill.update.short": {
		"Update the installed skills to this CLI's version",
		"Actualiza los skills instalados a la versión de este CLI",
		"Atualiza os skills instalados para a versão deste CLI",
	},
	"flag.force_skill": {
		"rewrite them even if the installed ones are newer",
		"los reescribe aunque los instalados sean más nuevos",
		"reescreve mesmo que os instalados sejam mais novos",
	},
	"skill.updated":        {"Updated %s (v%s)", "Actualizado %s (v%s)", "Atualizado %s (v%s)"},
	"skill.none_installed": {"There are no Rapibase skills installed: `rapibase skill install`", "No hay skills de Rapibase instalados: `rapibase skill install`", "Não há skills do Rapibase instalados: `rapibase skill install`"},
	"skill.up_to_date":     {"The %d installed skill(s) are up to date (v%s)", "Los %d skill(s) instalados están al día (v%s)", "Os %d skill(s) instalados estão em dia (v%s)"},
	"skill.installed":      {"Skills v%s for %s: %s (%s)", "Skills v%s para %s: %s (%s)", "Skills v%s para %s: %s (%s)"},
	"skill.kept_newer": {
		"Kept %s in %s: the installed one is newer",
		"Se mantuvo %s en %s: el instalado es más nuevo",
		"Mantido %s em %s: o instalado é mais novo",
	},
	"skill.agents_md": {
		"Added a line to AGENTS.md so the agent finds them",
		"Se agregó una línea a AGENTS.md para que el agente los encuentre",
		"Adicionada uma linha ao AGENTS.md para o agente encontrá-los",
	},
	"skill.git_excluded": {
		"Added to .git/info/exclude (only on this machine)",
		"Agregados a .git/info/exclude (solo en esta máquina)",
		"Adicionados ao .git/info/exclude (só nesta máquina)",
	},
	"skill.old_replaced": {
		"Replaced the old rapibase skill (%d place(s))",
		"Se reemplazó el skill viejo rapibase (%d lugar(es))",
		"Substituído o skill antigo rapibase (%d lugar(es))",
	},
	"skill.old_confirm": {
		"The old rapibase skill is installed (%d place(s)). Replace it with the new skills? Edits made to it are lost",
		"Está instalado el skill viejo rapibase (%d lugar(es)). ¿Lo reemplazo por los skills nuevos? Se pierden los cambios que le hayas hecho",
		"O skill antigo rapibase está instalado (%d lugar(es)). Substituí-lo pelos skills novos? As alterações feitas nele se perdem",
	},
	"skill.old_needs_yes": {
		"The old rapibase skill is installed (%d place(s)); replacing it deletes it. Run again with --yes",
		"Está instalado el skill viejo rapibase (%d lugar(es)); reemplazarlo lo borra. Vuelve a correrlo con --yes",
		"O skill antigo rapibase está instalado (%d lugar(es)); substituí-lo o apaga. Rode de novo com --yes",
	},
	"skill.old_kept": {
		"Kept the old rapibase skill (%d place(s)): the agent will see both guides until you remove it",
		"Se conservó el skill viejo rapibase (%d lugar(es)): el agente va a ver las dos guías hasta que lo borres",
		"Mantido o skill antigo rapibase (%d lugar(es)): o agente vai ver os dois guias até você apagá-lo",
	},
	"skill.old_retry": {
		"The skills were not installed. To replace the old one: rapibase skill install --yes",
		"No se instalaron los skills. Para reemplazar el viejo: rapibase skill install --yes",
		"Os skills não foram instalados. Para substituir o antigo: rapibase skill install --yes",
	},
	"cmd.skill.uninstall.short": {
		"Remove the Rapibase skills",
		"Quita los skills de Rapibase",
		"Remove os skills do Rapibase",
	},
	"cmd.skill.uninstall.long": {
		"Remove the Rapibase skills from this repo (every agent's folder, the old rapibase skill and the AGENTS.md line), or with --scope user from your home folder. It asks first; without a terminal it needs --yes.",
		"Quita los skills de Rapibase de este repo (la carpeta de cada agente, el skill viejo rapibase y la línea de AGENTS.md), o con --scope user de tu carpeta personal. Pregunta antes; sin terminal necesita --yes.",
		"Remove os skills do Rapibase deste repo (a pasta de cada agente, o skill antigo rapibase e a linha do AGENTS.md), ou com --scope user da sua pasta pessoal. Pergunta antes; sem terminal precisa de --yes.",
	},
	"flag.scope_remove": {
		"where to look: project or local (this repo) or user (your home folder)",
		"dónde buscar: project o local (este repo) o user (tu carpeta personal)",
		"onde procurar: project ou local (este repo) ou user (sua pasta pessoal)",
	},
	"skill.none_found": {
		"There are no Rapibase skills here",
		"No hay skills de Rapibase aquí",
		"Não há skills do Rapibase aqui",
	},
	"skill.found": {
		"Rapibase skills found (%d):",
		"Skills de Rapibase encontrados (%d):",
		"Skills do Rapibase encontrados (%d):",
	},
	"skill.uninstall_confirm": {
		"Remove them (%d)? Edits made to them are lost",
		"¿Los quito (%d)? Se pierden los cambios que les hayas hecho",
		"Removê-los (%d)? As alterações feitas neles se perdem",
	},
	"skill.uninstall_needs_yes": {
		"Removing the Rapibase skills (%d) deletes them. Run again with --yes",
		"Quitar los skills de Rapibase (%d) los borra. Vuelve a correrlo con --yes",
		"Remover os skills do Rapibase (%d) os apaga. Rode de novo com --yes",
	},
	"skill.uninstalled": {
		"Removed (%d). `rapibase skill install` puts them back",
		"Quitados (%d). `rapibase skill install` los vuelve a poner",
		"Removidos (%d). `rapibase skill install` os coloca de volta",
	},

	// --- mcp ---
	"cmd.mcp.short": {"The Rapibase MCP for your agent", "El MCP de Rapibase para tu agente", "O MCP do Rapibase para o seu agente"},
	"cmd.mcp.add.short": {
		"Register the MCP in your agent",
		"Registra el MCP en tu agente",
		"Registra o MCP no seu agente",
	},
	"cmd.mcp.add.long": {
		"Register the Rapibase MCP in your agent, without secrets: the agent logs in with OAuth the first time. Where the config is per repo it is pinned to the linked project.",
		"Registra el MCP de Rapibase en tu agente, sin secretos: el agente inicia sesión con OAuth la primera vez. Donde la config es por repo queda fijado al proyecto vinculado.",
		"Registra o MCP do Rapibase no seu agente, sem segredos: o agente faz login com OAuth na primeira vez. Onde a config é por repo fica fixado no projeto vinculado.",
	},
	"cmd.mcp.remove.short": {
		"Remove the Rapibase MCP from your agent",
		"Quita el MCP de Rapibase de tu agente",
		"Remove o MCP do Rapibase do seu agente",
	},
	"cmd.mcp.remove.long": {
		"Remove the Rapibase MCP servers of this folder: the repo's configs and Claude Code's local scope, older per-project ones included. With --scope user, the ones in your home folder, Codex's and Hermes'. Give a name to remove only that one. It asks first; without a terminal it needs --yes.",
		"Quita los servidores MCP de Rapibase de esta carpeta: las configs del repo y el scope local de Claude Code, incluidos los viejos por proyecto. Con --scope user, los de tu carpeta personal, Codex y Hermes. Con un nombre quita solo ese. Pregunta antes; sin terminal necesita --yes.",
		"Remove os servidores MCP do Rapibase desta pasta: as configs do repo e o scope local do Claude Code, incluindo os antigos por projeto. Com --scope user, os da sua pasta pessoal, do Codex e do Hermes. Com um nome remove só esse. Pergunta antes; sem terminal precisa de --yes.",
	},
	"flag.agent_remove": {
		"only this agent: claude, cursor, opencode, codex or hermes (repeatable; all by default)",
		"solo este agente: claude, cursor, opencode, codex o hermes (se puede repetir; todos por defecto)",
		"só este agente: claude, cursor, opencode, codex ou hermes (pode repetir; todos por padrão)",
	},
	"mcp.none_found": {
		"There's no Rapibase MCP here",
		"No hay MCP de Rapibase aquí",
		"Não há MCP do Rapibase aqui",
	},
	"mcp.found": {
		"Rapibase MCP servers found (%d):",
		"Servidores MCP de Rapibase encontrados (%d):",
		"Servidores MCP do Rapibase encontrados (%d):",
	},
	"mcp.remove_confirm": {
		"Remove them (%d)?",
		"¿Los quito (%d)?",
		"Removê-los (%d)?",
	},
	"mcp.remove_needs_yes": {
		"Removing the Rapibase MCP servers (%d) needs --yes",
		"Quitar los servidores MCP de Rapibase (%d) necesita --yes",
		"Remover os servidores MCP do Rapibase (%d) precisa de --yes",
	},
	"mcp.removed": {
		"Removed %s (%s)",
		"Quitado %s (%s)",
		"Removido %s (%s)",
	},
	"mcp.remove_manual": {
		"Couldn't remove it by itself; run or edit:",
		"No se pudo quitar solo; corre o edita:",
		"Não deu para remover sozinho; rode ou edite:",
	},
	"mcp.restart_after_remove": {
		"Restart your agent so it stops loading them",
		"Reinicia tu agente para que deje de cargarlos",
		"Reinicie o seu agente para que pare de carregá-los",
	},
	"mcp.legacy_found": {
		"%s already has %d older Rapibase MCP server(s) for this project:",
		"%s ya tiene %d MCP viejo(s) de Rapibase para este proyecto:",
		"%s já tem %d MCP antigo(s) do Rapibase para este projeto:",
	},
	"mcp.legacy_token": {
		"keeps a key in plain text",
		"guarda una clave en texto plano",
		"guarda uma chave em texto puro",
	},
	"mcp.legacy_confirm": {
		"Replace them with rapibase? If not, the agent gets the same tools twice",
		"¿Los reemplazo por rapibase? Si no, el agente tiene las mismas herramientas dos veces",
		"Substituí-los por rapibase? Se não, o agente fica com as mesmas ferramentas duas vezes",
	},
	"mcp.legacy_needs_yes": {
		"%s has older Rapibase MCP servers for this project; replacing them removes them. Run again with --yes",
		"%s tiene MCP viejos de Rapibase para este proyecto; reemplazarlos los quita. Vuelve a correrlo con --yes",
		"%s tem MCP antigos do Rapibase para este projeto; substituí-los os remove. Rode de novo com --yes",
	},
	"mcp.legacy_kept": {
		"Left the MCP of %s as it was: the new one wasn't added, so the tools don't show up twice",
		"Dejé el MCP de %s como estaba: no agregué el nuevo, así las herramientas no aparecen dos veces",
		"Deixei o MCP do %s como estava: não adicionei o novo, assim as ferramentas não aparecem duas vezes",
	},
	"mcp.legacy_retry": {
		"The MCP was not added. To replace the old ones: rapibase mcp add --yes",
		"No se agregó el MCP. Para reemplazar los viejos: rapibase mcp add --yes",
		"O MCP não foi adicionado. Para substituir os antigos: rapibase mcp add --yes",
	},
	"mcp.failed":  {"MCP for %s: %v", "MCP para %s: %v", "MCP para %s: %v"},
	"mcp.already": {"The MCP was already in %s", "El MCP ya estaba en %s", "O MCP já estava no %s"},
	"mcp.already_as": {
		"The MCP was already in %s, as %s",
		"El MCP ya estaba en %s, como %s",
		"O MCP já estava no %s, como %s",
	},
	"mcp.added": {"MCP added to %s (%s)", "MCP agregado a %s (%s)", "MCP adicionado ao %s (%s)"},
	"mcp.manual": {
		"To finish the MCP in %s, run or paste:",
		"Para terminar el MCP en %s, corre o pega:",
		"Para terminar o MCP no %s, rode ou cole:",
	},
	"mcp.login": {"Log the MCP in with %s", "Inicia la sesión del MCP con %s", "Faça login no MCP com %s"},
	"mcp.restart_claude": {
		"Restart Claude Code so it loads the MCP (claude --continue keeps the conversation); approve rapibase if it asks, and log in once from /mcp",
		"Reinicia Claude Code para que cargue el MCP (claude --continue mantiene la conversación); aprueba rapibase si te lo pide e inicia sesión una vez desde /mcp",
		"Reinicie o Claude Code para carregar o MCP (claude --continue mantém a conversa); aprove rapibase se ele pedir e faça login uma vez em /mcp",
	},
	"mcp.restart_cursor": {
		"Reload Cursor and enable rapibase in Settings → MCP",
		"Recarga Cursor y activa rapibase en Settings → MCP",
		"Recarregue o Cursor e ative rapibase em Settings → MCP",
	},
	"mcp.restart_opencode": {
		"Restart OpenCode so it loads the MCP",
		"Reinicia OpenCode para que cargue el MCP",
		"Reinicie o OpenCode para carregar o MCP",
	},
	"mcp.restart_codex": {
		"Start a new Codex session so it loads the MCP",
		"Abre una sesión nueva de Codex para que cargue el MCP",
		"Abra uma sessão nova do Codex para carregar o MCP",
	},
	"mcp.restart_hermes": {
		"Start a new Hermes session so it loads the MCP",
		"Abre una sesión nueva de Hermes para que cargue el MCP",
		"Abra uma sessão nova do Hermes para carregar o MCP",
	},
	"mcp.restart_other": {
		"Restart your agent so it loads the MCP",
		"Reinicia tu agente para que cargue el MCP",
		"Reinicie o seu agente para carregar o MCP",
	},

	// --- init ---
	"cmd.init.short": {
		"Set up this folder: account, project, .env, skills and MCP",
		"Prepara esta carpeta: cuenta, proyecto, .env, skills y MCP",
		"Prepara esta pasta: conta, projeto, .env, skills e MCP",
	},
	"cmd.init.long": {
		"Set up this folder in one go: logs in, creates or picks the project, links it, writes .env and installs the skills and the MCP for your agent. Each step can be skipped and running it again is safe.",
		"Prepara esta carpeta de una vez: inicia sesión, crea o elige el proyecto, lo vincula, escribe el .env e instala los skills y el MCP para tu agente. Cada paso se puede saltear y correrlo de nuevo es seguro.",
		"Prepara esta pasta de uma vez: faz login, cria ou escolhe o projeto, vincula, escreve o .env e instala os skills e o MCP para o seu agente. Cada passo pode ser pulado e rodar de novo é seguro.",
	},
	"flag.init_name": {
		"create a free project with this name",
		"crea un proyecto gratis con este nombre",
		"cria um projeto grátis com este nome",
	},
	"flag.no_env":   {"do not write .env", "no escribe el .env", "não escreve o .env"},
	"flag.no_skill": {"do not install the skills", "no instala los skills", "não instala os skills"},
	"flag.no_mcp":   {"do not register the MCP", "no registra el MCP", "não registra o MCP"},
	"flag.force_link": {
		"replace the folder's link without asking",
		"reemplaza el vínculo de la carpeta sin preguntar",
		"substitui o vínculo da pasta sem perguntar",
	},
	"init.app":     {"App", "App", "App"},
	"init.console": {"Console", "Consola", "Console"},
	"init.next":    {"Next:", "Lo que sigue:", "Próximos passos:"},
	"init.next_prompt": {
		"Ask your agent for what you want to build: it already knows how to use Rapibase.",
		"Pídele a tu agente lo que quieres construir: ya sabe usar Rapibase.",
		"Peça ao seu agente o que você quer construir: ele já sabe usar o Rapibase.",
	},
	"init.next_deploy": {
		"Publish with `rapibase deploy`; see everything with `rapibase status`.",
		"Publica con `rapibase deploy`; mira todo con `rapibase status`.",
		"Publique com `rapibase deploy`; veja tudo com `rapibase status`.",
	},
	"init.using_linked": {"Using %s (linked in %s)", "Usando %s (vinculado en %s)", "Usando %s (vinculado em %s)"},
	"init.need_name_new": {
		"you have no projects: pass --name to create a free one",
		"no tienes proyectos: pasa --name para crear uno gratis",
		"você não tem projetos: passe --name para criar um grátis",
	},
	"init.need_project": {
		"say which project to use (--project) or create one (--name)",
		"di qué proyecto usar (--project) o crea uno (--name)",
		"diga qual projeto usar (--project) ou crie um (--name)",
	},
	"init.create_new":  {"Create a new one", "Crear uno nuevo", "Criar um novo"},
	"init.ask_project": {"Which project?", "¿Qué proyecto?", "Qual projeto?"},
}

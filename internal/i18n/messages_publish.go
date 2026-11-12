package i18n

var publishMessages = map[string][3]string{
	// --- deploy (site) ---
	"cmd.deploy.short": {"Publish the site", "Publica el sitio", "Publica o site"},
	"cmd.deploy.long": {
		"Publish a static site (the build folder of the framework, or the one you pass). Each deploy is a release you can go back to with `rapibase deploys activate`.",
		"Publica un sitio estático (la carpeta del build del framework, o la que pases). Cada deploy es un release al que puedes volver con `rapibase deploys activate`.",
		"Publica um site estático (a pasta de build do framework, ou a que você passar). Cada deploy é um release ao qual você pode voltar com `rapibase deploys activate`.",
	},
	"flag.exclude": {
		"leave out files or folders (repeatable)",
		"deja afuera archivos o carpetas (se puede repetir)",
		"deixa de fora arquivos ou pastas (pode repetir)",
	},
	"flag.allow_dirty": {
		"publish even with uncommitted changes",
		"publica aunque haya cambios sin commitear",
		"publica mesmo com mudanças sem commit",
	},
	"flag.force_deploy": {
		"publish even if a GitHub Actions workflow already deploys",
		"publica aunque un workflow de GitHub Actions ya haga el deploy",
		"publica mesmo que um workflow do GitHub Actions já faça o deploy",
	},
	"deploy.has_workflow": {
		"%s already deploys this site from GitHub Actions: a manual deploy would be replaced on the next push",
		"%s ya publica este sitio desde GitHub Actions: un deploy manual quedaría pisado en el próximo push",
		"%s já publica este site pelo GitHub Actions: um deploy manual seria substituído no próximo push",
	},
	"deploy.has_workflow_hint": {
		"Push to the branch instead, or pass --force to publish anyway.",
		"Haz push a la rama, o pasa --force para publicar igual.",
		"Faça push para a branch, ou passe --force para publicar mesmo assim.",
	},
	"deploy.confirm_anyway": {"Publish anyway?", "¿Publicar igual?", "Publicar mesmo assim?"},
	"deploy.cancelled":      {"cancelled", "cancelado", "cancelado"},
	"deploy.dirty": {
		"there are uncommitted changes: the release would not match any commit",
		"hay cambios sin commitear: el release no coincidiría con ningún commit",
		"há mudanças sem commit: o release não corresponderia a nenhum commit",
	},
	"deploy.dirty_hint": {
		"Commit first, or pass --allow-dirty.",
		"Commitea primero, o pasa --allow-dirty.",
		"Faça commit antes, ou passe --allow-dirty.",
	},
	"deploy.secrets": {
		"%s has files that must not be published: %s",
		"%s tiene archivos que no se pueden publicar: %s",
		"%s tem arquivos que não podem ser publicados: %s",
	},
	"deploy.secrets_hint": {
		"Publish the build folder (dist, build, out…), or leave them out with --exclude.",
		"Publica la carpeta del build (dist, build, out…), o déjalos afuera con --exclude.",
		"Publique a pasta de build (dist, build, out…), ou deixe-os de fora com --exclude.",
	},
	"deploy.limit": {
		"the %s goes over the limit: %s (max %s)",
		"%s supera el límite: %s (máximo %s)",
		"%s passa do limite: %s (máximo %s)",
	},
	"deploy.no_index": {
		"%s has no index.html",
		"%s no tiene index.html",
		"%s não tem index.html",
	},
	"deploy.uploading": {
		"Uploading %d files (%s)…",
		"Subiendo %d archivos (%s)…",
		"Enviando %d arquivos (%s)…",
	},
	"deploy.done":    {"Published at %s", "Publicado en %s", "Publicado em %s"},
	"deploy.release": {"Release %s", "Release %s", "Release %s"},
	"deploy.no_dir":  {"the folder %s does not exist", "la carpeta %s no existe", "a pasta %s não existe"},
	"deploy.next_ssr": {
		"this Next app renders on the server: it is not a static site",
		"esta app de Next renderiza en el servidor: no es un sitio estático",
		"este app Next renderiza no servidor: não é um site estático",
	},
	"deploy.no_detect": {
		"could not tell which folder to publish: pass it (rapibase deploy ./dist)",
		"no se pudo saber qué carpeta publicar: pásala (rapibase deploy ./dist)",
		"não deu para saber qual pasta publicar: passe-a (rapibase deploy ./dist)",
	},
	"deploy.not_built": {
		"%s does not exist yet: build the site first",
		"%s todavía no existe: primero haz el build del sitio",
		"%s ainda não existe: faça o build do site antes",
	},
	"deploy.build_first": {"Run `%s`.", "Corre `%s`.", "Rode `%s`."},
	"deploy.detected":    {"Publishing %s (%s)", "Publicando %s (%s)", "Publicando %s (%s)"},

	// --- deploys ---
	"cmd.deploys.short": {
		"List the site's releases",
		"Lista los releases del sitio",
		"Lista os releases do site",
	},
	"cmd.deploys.activate.short": {
		"Go back to a release",
		"Vuelve a un release",
		"Volta para um release",
	},
	"deploys.none":      {"No deploys yet.", "Todavía no hay deploys.", "Ainda não há deploys."},
	"deploys.activated": {"Release %s is live", "El release %s quedó publicado", "O release %s está no ar"},

	// --- functions ---
	"cmd.functions.short": {
		"Server functions: endpoints, crons and workers",
		"Functions del servidor: endpoints, crons y workers",
		"Functions do servidor: endpoints, crons e workers",
	},
	"cmd.functions.deploy.short": {
		"Publish the functions folder",
		"Publica la carpeta functions",
		"Publica a pasta functions",
	},
	"cmd.functions.deploy.long": {
		"Publish the .ts and .js files of a folder (functions/ by default). Files on the server that are not in the folder stay, unless you pass --prune.",
		"Publica los archivos .ts y .js de una carpeta (functions/ por defecto). Los del servidor que no estén en la carpeta se quedan, salvo que pases --prune.",
		"Publica os arquivos .ts e .js de uma pasta (functions/ por padrão). Os do servidor que não estão na pasta ficam, a menos que você passe --prune.",
	},
	"cmd.functions.list.short": {"List the functions", "Lista las functions", "Lista as functions"},
	"cmd.functions.rm.short":   {"Delete a function file", "Borra un archivo de functions", "Apaga um arquivo de functions"},
	"flag.prune": {
		"delete the server files that are not in the folder",
		"borra los archivos del servidor que no están en la carpeta",
		"apaga os arquivos do servidor que não estão na pasta",
	},
	"fn.no_dir": {
		"there is no %s folder: pass the folder of the functions",
		"no hay una carpeta %s: pasa la carpeta de las functions",
		"não há uma pasta %s: passe a pasta das functions",
	},
	"fn.bad_name": {
		"invalid file name %q: letters, numbers, . _ - and .ts or .js",
		"nombre de archivo inválido %q: letras, números, . _ - y .ts o .js",
		"nome de arquivo inválido %q: letras, números, . _ - e .ts ou .js",
	},
	"fn.none": {
		"%s has no .ts or .js files",
		"%s no tiene archivos .ts ni .js",
		"%s não tem arquivos .ts nem .js",
	},
	"fn.too_many": {
		"%d files: the limit is 50 per project",
		"%d archivos: el límite es 50 por proyecto",
		"%d arquivos: o limite é 50 por projeto",
	},
	"fn.uploading": {"Publishing %d file(s)…", "Publicando %d archivo(s)…", "Publicando %d arquivo(s)…"},
	"fn.prune_needs_yes": {
		"--prune would delete %s: confirm with --yes",
		"--prune borraría %s: confirma con --yes",
		"--prune apagaria %s: confirme com --yes",
	},
	"fn.prune_confirm": {
		"Delete from the server: %s?",
		"¿Borrar del servidor: %s?",
		"Apagar do servidor: %s?",
	},
	"fn.compile_errors": {
		"%d function file(s) did not compile",
		"%d archivo(s) de functions no compilaron",
		"%d arquivo(s) de functions não compilaram",
	},
	"fn.removed":  {"Deleted %s", "Borrado %s", "Apagado %s"},
	"fn.deployed": {"Published %d file(s): %s", "Publicados %d archivo(s): %s", "Publicados %d arquivo(s): %s"},

	// --- services ---
	"cmd.services.short": {
		"Your own services, in any language",
		"Tus servicios propios, en cualquier lenguaje",
		"Seus próprios serviços, em qualquer linguagem",
	},
	"services.none": {"No services yet.", "Todavía no hay servicios.", "Ainda não há serviços."},
	"cmd.services.deploy.short": {
		"Publish a service (Node, Bun, Python or a binary)",
		"Publica un servicio (Node, Bun, Python o un binario)",
		"Publica um serviço (Node, Bun, Python ou um binário)",
	},
	"cmd.services.deploy.long": {
		"Publish a service: the folder (Node, Bun or Python, detected by its files, or a rapibase.json) or a Linux x86-64 binary with --binary. It is built and started before answering; if it fails, the previous release keeps serving.",
		"Publica un servicio: la carpeta (Node, Bun o Python, detectado por sus archivos, o un rapibase.json) o un binario Linux x86-64 con --binary. Se construye y arranca antes de responder; si falla, sigue sirviendo el release anterior.",
		"Publica um serviço: a pasta (Node, Bun ou Python, detectado pelos arquivos, ou um rapibase.json) ou um binário Linux x86-64 com --binary. É construído e iniciado antes de responder; se falhar, o release anterior continua no ar.",
	},
	"cmd.services.restart.short":  {"Restart a service", "Reinicia un servicio", "Reinicia um serviço"},
	"cmd.services.activate.short": {"Go back to a release of a service", "Vuelve a un release de un servicio", "Volta para um release de um serviço"},
	"cmd.services.rm.short":       {"Delete a service", "Borra un servicio", "Apaga um serviço"},
	"flag.binary": {
		"Linux x86-64 binary to run (the folder adds files it reads)",
		"binario Linux x86-64 a correr (la carpeta suma archivos que lee)",
		"binário Linux x86-64 a rodar (a pasta soma arquivos que ele lê)",
	},
	"flag.svc_path": {
		"URL path where it is mounted (/api, or / for the whole site)",
		"path de la URL donde se monta (/api, o / para todo el sitio)",
		"caminho da URL onde é montado (/api, ou / para o site todo)",
	},
	"flag.svc_subdomain": {
		"also serve it on its own subdomain",
		"también lo sirve en un subdominio propio",
		"também o serve em um subdomínio próprio",
	},
	"flag.svc_domain": {"custom domain for the service", "dominio propio para el servicio", "domínio próprio para o serviço"},
	"svc.bad_name": {
		"invalid service name %q: lowercase letters and numbers, starts with a letter, up to 20",
		"nombre de servicio inválido %q: minúsculas y números, empieza con letra, hasta 20",
		"nome de serviço inválido %q: minúsculas e números, começa com letra, até 20",
	},
	"svc.no_binary": {"there is no file %s", "no existe el archivo %s", "o arquivo %s não existe"},
	"svc.bad_binary_windows": {
		"%s is a Windows binary: the server runs Linux x86-64",
		"%s es un binario de Windows: el servidor corre Linux x86-64",
		"%s é um binário do Windows: o servidor roda Linux x86-64",
	},
	"svc.bad_binary_macos": {
		"%s is a macOS binary: the server runs Linux x86-64",
		"%s es un binario de macOS: el servidor corre Linux x86-64",
		"%s é um binário do macOS: o servidor roda Linux x86-64",
	},
	"svc.bad_binary_linux": {
		"%s is a Linux binary for another architecture: the server runs x86-64",
		"%s es un binario Linux de otra arquitectura: el servidor corre x86-64",
		"%s é um binário Linux de outra arquitetura: o servidor roda x86-64",
	},
	"svc.bad_binary_unknown": {
		"%s is not a Linux x86-64 binary",
		"%s no es un binario Linux x86-64",
		"%s não é um binário Linux x86-64",
	},
	"svc.secrets": {
		"the folder has files that must not be uploaded: %s",
		"la carpeta tiene archivos que no se pueden subir: %s",
		"a pasta tem arquivos que não podem ser enviados: %s",
	},
	"svc.secrets_hint": {
		"Move the secrets to `rapibase config set` (they reach the service as variables).",
		"Pasa los secretos a `rapibase config set` (le llegan al servicio como variables).",
		"Passe os segredos para `rapibase config set` (chegam ao serviço como variáveis).",
	},
	"svc.empty": {"%s is empty", "%s está vacía", "%s está vazia"},
	"svc.deploying": {
		"Publishing %s (%d files, %s): building and starting…",
		"Publicando %s (%d archivos, %s): construyendo y arrancando…",
		"Publicando %s (%d arquivos, %s): construindo e iniciando…",
	},
	"svc.failed": {
		"%s failed (%s): %s. The previous release keeps serving",
		"%s falló (%s): %s. Sigue sirviendo el release anterior",
		"%s falhou (%s): %s. O release anterior continua no ar",
	},
	"svc.ready":     {"%s is live at %s", "%s está publicado en %s", "%s está no ar em %s"},
	"svc.restarted": {"Restarted %s", "Reiniciado %s", "Reiniciado %s"},
	"svc.activated": {"%s is back on release %s", "%s volvió al release %s", "%s voltou para o release %s"},
	"svc.confirm_rm": {
		"Delete the service %s and its releases?",
		"¿Borrar el servicio %s y sus releases?",
		"Apagar o serviço %s e seus releases?",
	},
	"svc.rm_needs_yes": {
		"deleting %s needs --yes",
		"borrar %s necesita --yes",
		"apagar %s precisa de --yes",
	},
	"svc.removed": {"Deleted the service %s", "Servicio %s borrado", "Serviço %s apagado"},
	"cmd.services.update.short": {
		"Change where a service is served (path, subdomain, domain)",
		"Cambia dónde se sirve un servicio (path, subdominio, dominio)",
		"Muda onde um serviço é servido (caminho, subdomínio, domínio)",
	},
	"svc.update_nothing": {
		"nothing to change: pass --path, --subdomain or --custom-domain",
		"nada que cambiar: pasa --path, --subdomain o --custom-domain",
		"nada para mudar: passe --path, --subdomain ou --custom-domain",
	},
	"svc.updated": {"Updated the routing of %s", "Se actualizó el ruteo de %s", "Roteamento de %s atualizado"},
}

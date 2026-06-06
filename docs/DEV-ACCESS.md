# NebulaCloud — como acessar em dev (Windows / Docker Desktop)

Só `http://localhost:3000` abrir não significa que a stack está quebrada: o **frontend** expõe a porta 3000 direto; o resto usa **Traefik (porta 80)** ou **portas publicadas** no override.

## URLs que funcionam sem `hosts`

| Serviço | URL |
|---------|-----|
| Dashboard (Next.js) | http://localhost:3000 |
| API | http://localhost:8081 |
| API health | http://localhost:8081/healthz |
| API OpenAPI | http://localhost:8081/api/v1/openapi.yaml |
| Prometheus | http://localhost:9090 |
| Loki | http://localhost:3100 |
| Grafana | http://localhost:3001 (admin / admin) |
| Registry | http://localhost:5000/v2/ |
| Traefik dashboard | http://localhost:8080 |

O frontend em dev chama a API em **`http://localhost:8081`** (variável `NEXT_PUBLIC_API_URL` no override).

## URLs com `*.nebula.localhost` (porta 80 → Traefik)

Requer entradas no ficheiro hosts (ou resolução automática de `*.localhost` no teu Windows):

```text
127.0.0.1  api.nebula.localhost app.nebula.localhost grafana.nebula.localhost
           traefik.nebula.localhost registry.nebula.localhost
```

(`make hosts` imprime a mesma linha.)

| Host | URL |
|------|-----|
| API | http://api.nebula.localhost/healthz |
| App | http://app.nebula.localhost |
| Grafana | http://grafana.nebula.localhost |
| Traefik | http://traefik.nebula.localhost |

Rotas da plataforma em `deployments/traefik/dynamic/dynamic.yml`; apps do utilizador em `deployments/traefik/dynamic/nebula-*.yml` (escritos pelo `runtime-agent`, mesmo diretório — o Traefik não lê subpastas). Isto evita depender do Docker provider do Traefik (falha comum no Docker Desktop).

## Depois de mudar Traefik / API

```powershell
docker compose up -d --force-recreate traefik api frontend
```

## Verificar

```powershell
Invoke-WebRequest http://localhost:8081/healthz -UseBasicParsing
Invoke-WebRequest http://api.nebula.localhost/healthz -UseBasicParsing
```

Ambos devem devolver **200**.

## Conectar repositório GitHub e implantar

Não existe um botão global “Importar do GitHub” no dashboard. O fluxo passa por **Projetos**:

1. Abrir o dashboard: http://localhost:3000 (ou http://app.nebula.localhost)
2. Fazer login ou criar conta
3. Barra lateral → **Projetos** (`/projects`)
4. Se não houver organização: **Criar org padrão**
5. **Novo projeto** ou **Criar projeto demo**
6. Abrir o card do projeto → no topo à direita: **Conectar repositório** (ícone GitHub)
   - Alternativa: **Configurar** → campo URL do repositório
   - Opcional: **Procurar no GitHub** (OAuth) para escolher um repo da lista
7. Na aba **Visão geral**: **Adicionar serviço web**
8. **Implantar agora** no cabeçalho do projeto (ou **Implantar** em cada serviço)

### Webhook (deploy automático no push)

No diálogo **Conectar repositório**, copie o endpoint:

- Dev direto: `http://localhost:8081/api/v1/webhooks/github`
- Via Traefik: `http://api.nebula.localhost/api/v1/webhooks/github`

Configure no GitHub (Settings → Webhooks) com secret `NEBULA_GITHUB_APP_WEBHOOK_SECRET` e evento **push**.

### OAuth GitHub (lista de repositórios)

Defina no `.env`:

- `NEBULA_GITHUB_APP_CLIENT_ID` e `NEBULA_GITHUB_APP_CLIENT_SECRET`
- `NEBULA_GITHUB_OAUTH_REDIRECT_URL` — callback da API (ex. `http://localhost:8081/api/v1/auth/github/callback`)
- `NEBULA_APP_URL` — URL do frontend (ex. `http://localhost:3000`)

No login ou no diálogo de conexão, use **Continuar com GitHub** / **Procurar no GitHub**.

## Stack mínima vs deploy completo

| Objetivo | Serviços necessários |
|----------|----------------------|
| Só dashboard + login + CRUD | `postgres`, `redis`, `api`, `frontend` |
| **Deploy que sobe container** | Acima + `build-worker`, `runtime-agent`, `registry`, `traefik` |

```powershell
docker compose up -d --build
docker compose ps
# build-worker e runtime-agent devem estar healthy
```

### Erro `membership not found` ao criar projeto

Causa típica: ID de organização antigo no **localStorage** (`nebula_org`) após `docker compose down -v` ou troca de utilizador.

**Correção rápida:** DevTools → Application → Local Storage → apagar `nebula_org` → recarregar → login (cria org automaticamente) → criar projeto.

O dashboard também re-sincroniza a org selecionada com `GET /organizations`.

### Erro `docker push` / `registry.nebula.localhost-5000: no such host`

O build corre no **Docker do host** (socket montado no `build-worker`). A referência da imagem deve usar um registry acessível do host, tipicamente **`localhost:5000`** (porta publicada do serviço `registry`).

| Sintoma | Causa | Correção |
|---------|--------|----------|
| `registry.nebula.localhost-5000` (hífen antes de 5000) | Bug antigo: `:` na URL do registry era sanitizado | Atualize API + faça novo deploy |
| `lookup registry.nebula.localhost-5000` | Hostname inválido (ver acima) | Idem |
| HTTPS na porta 443 para registry local | Docker trata host:port como registry remoto | Marque registry como inseguro no Docker Desktop |

**Docker Desktop (Windows):** Settings → Docker Engine → inclua:

```json
"insecure-registries": ["localhost:5000", "registry.nebula.localhost:5000"]
```

Reinicie o Docker, depois:

```powershell
docker compose up -d --force-recreate api build-worker runtime-agent registry
```

Confirme o registry: `Invoke-WebRequest http://localhost:5000/v2/ -UseBasicParsing` → 200.

### Deploy falhou / logs vazios no dashboard

**Na UI (após atualização):** abra **Implantações** → clique na linha → o painel mostra `error_message` em vermelho e o **log de build/deploy** (histórico Redis + ao vivo durante o build).

**Causas frequentes**

| Sintoma | O que verificar |
|---------|-----------------|
| Status `failed` sem detalhe antigo | Atualize API/frontend; a API agora expõe `error_message` no JSON do deployment. |
| Log vazio no painel | `build-worker` e `runtime-agent` a correr? Mesmo Redis que a API? |
| Fica em `building` para sempre | Worker parado: `docker compose ps` → `build-worker` healthy. |
| Fica em `deploying` para sempre | Job `deploy.run` foi consumido pelo `build-worker` (fila `critical` antiga) em vez do `runtime-agent`. Atualize workers (`docker compose up -d --force-recreate build-worker runtime-agent`), marque o deployment como `failed` no SQL e dispare deploy de novo. Nos logs do build-worker aparece `Retry exhausted` sem `deploy.start` no runtime-agent. |
| Erro de clone / branch | Branch no projeto (`main` vs `master`); repo privado sem credenciais. |
| Erro `docker build` | Repo sem `Dockerfile` na raiz e buildpack não detectado. |
| Erro `docker push` | Serviço `registry` up; worker na rede `nebula_platform`. |
| Erro `docker pull` / `docker run` | `runtime-agent` com socket Docker; Traefik/rede `nebula_platform`. |
| **404** na URL do app (`http://{svc}.{proj}.nebula.localhost`) | O `runtime-agent` grava `deployments/traefik/dynamic/nebula-*.yml` e verifica o router no Traefik. **Redeploy** após `docker compose up -d --force-recreate traefik runtime-agent build-worker`. Host deve ter ponto (`nebula.localhost`). Se o ficheiro YAML existe mas ainda 404: `docker compose restart traefik`. Script: `.\scripts\verify-deploy-route.ps1 -HostHeader web-xxx.app-yyy.nebula.localhost` |

**Debug rápido**

```powershell
docker compose ps
docker compose logs -f build-worker runtime-agent
.\scripts\verify-deploy-route.ps1 -HostHeader "<service>.<project>.nebula.localhost"
```

**Bootstrap após `docker compose down -v`**

1. `docker compose up -d --build` (inclui traefik, build-worker, runtime-agent, registry).
2. Criar projeto + repo + **novo deploy** (não reutilizar URLs antigas).
3. Confirmar `deployments/traefik/dynamic/nebula-*.yml` com `Host(\`…nebula.localhost\`)` e porta correta (ex. 3000 para welcome-to-docker).
4. E2E opcional: `.\scripts\e2e-smoke.ps1 -WaitForDeploy`

SQL (Postgres):

```sql
SELECT id, status, error_message, created_at
FROM deployments
ORDER BY created_at DESC
LIMIT 5;
```

API:

```powershell
# substitua {deploymentId} e o token
Invoke-WebRequest "http://localhost:8081/api/v1/deployments/{deploymentId}/build-logs" `
  -Headers @{ Authorization = "Bearer <token>" } -UseBasicParsing
```

### Smoke test (API)

Com a API em `http://localhost:8081`:

```powershell
.\scripts\e2e-smoke.ps1
```

Com repositório público que tenha **Dockerfile** na raiz (recomendado para deploy completo):

```powershell
.\scripts\e2e-smoke.ps1 -RepoUrl "https://github.com/vercel/next.js/tree/canary/examples/hello-world"
```

**Repositórios de teste recomendados**

| Repo | Notas |
|------|--------|
| [vercel/next.js `examples/hello-world`](https://github.com/vercel/next.js/tree/canary/examples/hello-world) | Dockerfile na pasta do exemplo — use URL do subdiretório ou clone manual; para smoke, prefira repos com Dockerfile na **raiz**. |
| [tiangolo/uvicorn-gunicorn-fastapi-docker](https://github.com/tiangolo/uvicorn-gunicorn-fastapi-docker) | Dockerfile na raiz, build rápido, bom para Teste C. |
| [docker/welcome-to-docker](https://github.com/docker/welcome-to-docker) | Imagem mínima, ideal para validar build-worker → registry → runtime-agent. |

**Evitar:** `octocat/Hello-World` — não tem Dockerfile nem buildpack detectável; o build falha.

Exemplo Teste C (deploy até container):

```powershell
.\scripts\e2e-smoke.ps1 -RepoUrl "https://github.com/docker/welcome-to-docker" -Branch main
docker compose logs -f build-worker runtime-agent
```

Depois, na UI: aba **Implantações** → status deve evoluir de `building` para `running` (ou `failed` com mensagem no log).

Teste Go de membership (requer Postgres):

```powershell
$env:NEBULA_TEST_DSN = "postgres://nebula:nebula@localhost:5432/nebula?sslmode=disable"
go test ./internal/modules/projects/interfaces/ -run TestOrgProjectMembershipHTTP -count=1
```

### cAdvisor (logs no container)

Mensagens como `failed to identify the read-write layer ID` / `layerdb/mounts/.../mount-id` vêm do **cAdvisor** a monitorizar containers no Docker Desktop/WSL2 — **não são erro da API nem do frontend**. São inofensivas.

Em dev, o cAdvisor fica desligado por defeito (`docker-compose.override.yml`, profile `metrics`). Para métricas:

```powershell
docker compose --profile metrics up -d cadvisor
```

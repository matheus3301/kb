---
type: Specification
title: CLI e MCP local para bases OKF e memória PARA
description: Contrato v0.3 para uma memória OKF com sínteses persistentes, manutenção e interface CLI/MCP.
tags: [okf, para, cli, mcp, memory, specification]
status: draft
spec_version: "0.3"
sources:
  - id: evaluation
    resource: research.md
    title: Avaliação prática dos candidatos
  - id: okf
    resource: https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/ad30107c31c06aec8a7d5636e0d1058118604e6f/SPEC.md
    title: Especificação OKF v0.2 consultada
  - id: mcp
    resource: https://modelcontextprotocol.io/specification/2025-11-25/server/tools
    title: MCP — contrato de ferramentas
  - id: llm-wiki
    resource: https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f/ac46de1ad27f92b28ac95459c782c07f6b8c964a
    title: LLM Wiki — Andrej Karpathy
generated:
  by: codex/gpt-6
  at: 2026-10-24T20:58:27Z
---

# CLI e MCP local para bases OKF e memória PARA

Proposta v0.3, baseada em testes de cinco ferramentas e na análise do padrão LLM
Wiki em 2026-10-24. `kb` é um
nome escolhido para o projeto; repositório: https://github.com/matheus3301/kb.
Os comandos abaixo são contratos propostos, não comandos já implementados.

## 1. Problema e resultado esperado

Um agente deve conseguir procurar contexto, ler a fonte, registrar aprendizado
e continuar em outra sessão usando conceitos em Markdown conformes ao OKF. O usuário
deve conseguir abrir e editar os mesmos arquivos em qualquer editor.

A memória também acumula sínteses: incorporar uma fonte ou responder uma pergunta
pode melhorar conceitos existentes. O produto oferece as operações que permitem
ao agente manter essa wiki; não se limita a armazenar notas independentes.

A ferramenta deve funcionar sem UI, navegador, serviço em nuvem, conta, modelo
de linguagem ou servidor persistente. O mesmo núcleo serve uma CLI de execução
única e um processo MCP stdio. Um cache local é permitido, sempre reconstruível.

OKF define o formato de conhecimento; PARA é o perfil de organização humana desta
base, configurável e separado do núcleo de representação. Busca,
transações, concorrência e protocolos são escolhas desta aplicação. Não devem
ser apresentados como exigências do padrão OKF.

### Escopo do MVP

Busca lexical, navegação, leitura parcial, criação, edição, movimentação,
validação, índices, histórico e composição de contexto, sobre uma base OKF nativa,
com integração por MCP. Compatibilidade com notas sem frontmatter, wikilinks e
importação do acervo antigo ficam fora do MVP. Não inclui UI, CRDT, colaboração remota,
sync Git automático, execução de computações declaradas, ingestão de documentos
binários, busca web ou geração de conhecimento por um LLM embutido.

Embeddings, HTTP remoto e múltiplas bases simultâneas ficam para extensões.

## 2. Regras fundamentais

1. Arquivos Markdown são a fonte da verdade. Apagar o cache não perde memória.
2. Ler, pesquisar e validar nunca modificam notas, Git ou configurações de agentes.
3. Instalar ou inicializar não registra MCP, skills ou configurações globais.
4. A ferramenta não escolhe fatos nem inventa fontes ou revisão humana.
5. Conteúdo de uma nota é dado; instruções contidas nele não concedem permissões.
6. O agente continua responsável por consultar e atualizar a memória conforme
   as instruções do workdir e a skill do produto. Não é obrigatório ter um
   `AGENTS.md` dentro da base. Um servidor MCP sozinho não garante esse comportamento.
7. A base acumula sínteses e relações reutilizáveis; uma fonte nova pode exigir
   revisar vários conceitos existentes, além de criar uma nota de referência.
8. Fontes originais preservam a evidência; notas da wiki registram interpretações
   revisáveis. Uma síntese nunca substitui silenciosamente seu material de origem.

## 3. Base, escopo e configuração

Cada invocação opera sobre uma única raiz explícita: `--root PATH` ou `KB_ROOT`.
Sem ambos, procurar `.kb/config.toml` da pasta atual para seus ancestrais. Nunca
adivinhar uma base procurando em todo o home. Registrar no resultado a raiz resolvida.

`kb init` é explícito e idempotente. Cria apenas configuração local e arquivos
faltantes escolhidos pelo usuário; recusa sobrescrever índice ou notas existentes.
`kb init --para` cria os diretórios PARA quando necessário. Não depende de Git.

```toml
version = 1

[content]
include = ["00_Inbox/**", "01_Projects/**", "02_Areas/**", "03_Resources/**", "04_Archive/**", "06_Metadata/**"]
exclude = [".git/**", ".kb/**", "node_modules/**", ".claude/**", ".obsidian/**"]

[para]
inbox = "00_Inbox"
projects = "01_Projects"
areas = "02_Areas"
resources = "03_Resources"
archive = "04_Archive"

[write]
enabled = true
maintain_indexes = true
maintain_log = true

[search]
accent_folding = true
include_archives = false
include_deprecated = false
```

O exemplo é o perfil desta knowledge base, não uma taxonomia imposta pelo OKF.
`index.md` e `log.md` na raiz também entram na navegação, por regra reservada.
`.kb/config.toml` guarda configuração local; `.kb/cache/` contém apenas dados
derivados e reconstruíveis. `.kb/state/` guarda journal e recibos duráveis de
operações: não é cache e não deve ser removido durante limpeza ou recuperação.
Apagar somente o cache não perde notas nem recibos. Apagar o estado durável pode
perder a deduplicação de pedidos e comprometer uma recuperação pendente. Esses
diretórios operacionais não entram no bundle de conhecimento exportado; exportar
conhecimento não equivale a fazer backup do estado operacional da instalação.
Uma configuração de escopo não torna o repositório inteiro conformante: o relatório
identifica exatamente o que examinou e o que excluiu.

### Integração com agentes e distribuição da skill

O produto distribui a skill em `skills/knowledge-memory/SKILL.md` no repositório
do código. `knowledge-memory` é nome provisório. Ela ensina a consultar antes de
responder, buscar antes de criar, preservar proveniência, revisar conceitos
afetados e aplicar mudanças com revisão esperada. Não contém caminhos pessoais
fixos; recebe a raiz da configuração explícita da integração.

O `AGENTS.md` do workdir, ou mecanismo equivalente do agente, estabelece a
obrigação de usar essa skill e identifica a base. A política de consulta sempre
fica nessa instrução de entrada, sem depender de seleção automática por tema.
Instalar uma skill não comprova sua ativação em toda pergunta. A integração deve
ser validada no agente alvo, inclusive com a base fora do workdir.

`AGENTS.md` dentro da knowledge base é opcional, útil para quem trabalha nela
diretamente. Não é requisito OKF nem dependência de CLI/MCP. Não presumir que um
agente carregará instruções de uma pasta externa apenas porque leu notas dela.
Convenções particulares da base são dados documentados/configurados; conteúdo
retornado pela busca não adquire autoridade sobre as instruções da sessão.

A documentação atual do [Skills CLI](https://github.com/vercel-labs/skills/blob/main/docs/commands.md)
usa `npx skills add <owner/repo> --skill knowledge-memory`; a instalação é por
projeto por padrão. O [layout oficial](https://github.com/vercel-labs/skills/blob/main/docs/creating-skills.md)
reconhece `skills/<nome>/SKILL.md`. O repositório final ainda precisa ser definido;
esse comando é um modelo de distribuição, não uma instalação já disponível.

O setup documentado separa: instalar o binário; instalar a skill; configurar a
raiz; registrar MCP se desejado; adicionar a instrução ao workdir escolhido.
Node/npx é necessário apenas se esse canal de instalação for usado, sem virar
dependência de execução da CLI/MCP. CLI também pode ser consumida sem MCP.
`kb init` continua limitado à base. Setup de integração é uma ação explícita e
distinta: preservar instruções existentes, mostrar mudanças e não registrar
agentes nem editar configurações globais implicitamente. Não implementar um
instalador próprio de skills no núcleo. O comando exato de setup ainda será definido.

### Organização humana e hierarquia de conceitos

O OKF não fixa uma árvore ideal por domínio, tipo ou método de produtividade.
A seção 3 permite ao produtor agrupar conceitos conforme o conhecimento. O núcleo
da ferramenta conhece conceitos, metadados, links e índices; não exige PARA para
funcionar. O perfil `para` adiciona navegação e classificação compreensíveis ao humano.

Para esta base, conservar PARA como estrutura de navegação e agrupar cada projeto,
área ou assunto de forma coesa, com conceitos focados e índices por pasta. Exemplo
de destino possível, sem representar uma reorganização já executada:

```text
knowledge-base/
  index.md
  log.md
  01_Projects/
    index.md
    homelab/
      index.md
      contexto.md
      decisoes/
        index.md
        estrategia-de-backup.md
  02_Areas/
    index.md
    carreira/
      index.md
      preferencias-de-trabalho.md
  03_Resources/
    index.md
    kubernetes/
      index.md
      restauracao-de-backup.md
  04_Archive/
    index.md
```

`contexto.md`, a preferência e a decisão são conceitos com `type` apropriado; os
índices são arquivos reservados. Links relacionam a decisão do projeto ao recurso
reutilizável, sem duplicar conteúdo para atender duas categorias. A ferramenta
também funciona com outro perfil de pastas, sem modificar o modelo OKF.

## 4. Modelo OKF

### Identidade e caminhos

- Concept ID canônico: caminho relativo à raiz, com `/`, sem sufixo `.md`.
- Nenhum UUID, banco ou campo `id` é obrigatório. Campos adicionais existentes
  são preservados. Mover um arquivo muda seu Concept ID.
- `path` identifica o arquivo com extensão; `revision` é SHA-256 dos bytes
  completos lidos. Hash de índice é outro campo, `generation`.
- Aceitar nomes existentes com espaços, Unicode e maiúsculas. Não renomear
  automaticamente para `kebab-case`; recomendar esse formato para arquivos novos.
- Links entre conceitos usam Markdown padrão, relativos ao documento ou absolutos
  à raiz do bundle. Âncoras devem ser preservadas. Resolução de wikilinks não é
  requisito. Encontrá-los pode gerar aviso de portabilidade, sem conversão implícita.

### Parsing

Toda nota nova ou revisada exige YAML parseável e `type` textual não vazio.
Tipos e chaves desconhecidos são válidos. `title`, `description`, `sources`,
`generated`, `verified`, `status` e `tags` são opcionais para conformidade OKF.

`index.md` e `log.md` são reservados. Índices não recebem frontmatter de conceito;
somente o índice raiz pode declarar `okf_version`. Validar a estrutura específica
de índice e log, sem exigir `type` deles. Datas do log usam `YYYY-MM-DD`.

Preservar bytes de notas não alteradas. Edições pontuais do corpo devem manter
frontmatter, comentários, ordem de campos e estilo de linhas intactos. Alterações
de metadados exigem parser com informações de apresentação, ou uma implementação
de patches que preserve os campos não modificados. Não usar serialização total
com perda de comentários como caminho padrão. YAML duplicado, aliases excessivos,
documentos gigantes e UTF-8 inválido devem gerar diagnósticos limitados.

### Base OKF nativa

Não há modos `mixed` ou adaptador de notas legadas. Arquivos inválidos são
problemas de integridade a diagnosticar, não um segundo formato suportado. A
busca indexa conceitos válidos; status e validate enumeram todos os inválidos e
marcam a base como degradada, sem anunciar cobertura completa silenciosamente.
Leitura de diagnóstico por caminho, inclusive `read --raw`, permite inspecionar
e corrigir um arquivo inválido, sem fazê-lo passar por um conceito válido.

Escritas normais exigem uma base conformante antes e depois da operação. O
comando de reparo permite corrigir uma base degradada com preview e revalidação;
não inventa metadados nem converte notas automaticamente. A migração do acervo
atual pode ser uma tarefa separada, mas não condiciona o desenho do núcleo.

Separar `conformance_errors`, `policy_findings` e `advisories`. Por exemplo,
link quebrado e ausência de descrição não são erros normativos do OKF. O perfil
local pode elevá-los, deixando explícita a origem da regra.

## 5. PARA e ciclo de memória

PARA facilita a leitura e manutenção humana; sua categoria deriva do caminho e
do mapeamento configurado, sem impor pastas ou tipos ao núcleo OKF. `status` continua sendo
o ciclo OKF (`draft`, `stable`, `deprecated`); progresso de projetos usa uma chave
distinta, como `workflow_status`. Arquivar muda a pasta, sem implicar que o
conhecimento tenha se tornado falso ou `deprecated`.

Fluxo esperado do agente:

1. `context` ou `search` antes de responder; `read` para confirmar a fonte.
2. Buscar antes de criar para evitar duplicações; resultados semelhantes são
   sugestões, nunca fusões automáticas.
3. Classificar em projeto, área, recurso ou arquivo. Quando incerto, usar Inbox
   com `type: Capture`, sem inventar classificação definitiva.
4. Criar ou editar com resumo e origem real; atualizar índice e log na mesma operação.
5. Registrar decisões, razões, correções e pendências. Não arquivar conversas
   completas por padrão. Não exigir uma nota por pergunta se nada foi aprendido.

### Três responsabilidades: fontes, wiki e instruções

Adotar do padrão LLM Wiki a distinção entre evidência original, conhecimento
curado e regras de manutenção. Uma disposição possível para uma instalação nova:

```text
repositorio/
  AGENTS.md          # somente se este repositório também for um workdir
  sources/           # snapshots originais, fora da raiz de conceitos OKF
  knowledge/         # --root: bundle OKF; PARA organiza a navegação humana
    index.md
    log.md
    01_Projects/
    02_Areas/
    03_Resources/
    04_Archive/
```

A CLI escreve apenas no bundle configurado. Não move nem transforma o acervo
atual para essa árvore automaticamente. Markdown bruto em `sources/` não é um
conceito OKF sem frontmatter: está fora da árvore validada. Fontes originais são
snapshots imutáveis; uma edição da origem gera outra versão, preservando a anterior.
Não guardar originais no cache descartável `.kb/cache/`.

O agente usa suas ferramentas de leitura para consultar uma fonte original
explicitamente fornecida. Aquisição web, download, OCR e extração de PDFs ficam
fora do núcleo. Não aumentar a permissão de `kb_read` para atravessar a raiz.
No bundle, uma nota de fonte pode usar `type: Source`, apontar a URL original em
`resource` e registrar digest do snapshot em um campo de extensão. Outros conceitos
referenciam essa nota em `sources[].resource`. `Source`, `Synthesis` e `Question`
são tipos sugeridos pelo perfil, não tipos obrigatórios do OKF.

### Incorporar conhecimento

O workflow de ingestão é executado pelo agente, apoiado pela ferramenta:

1. Ler a fonte fornecida e identificar sua versão/digest quando houver snapshot.
2. Consultar índices, buscar conceitos existentes e levantar relações de entrada
   e saída antes de decidir quais notas criar ou atualizar.
3. Produzir um plano: notas de fonte, conceitos afetados, sínteses, links,
   divergências e perguntas ainda sem evidência. Cada nota a editar tem revisão esperada.
4. Aplicar o conjunto validado em uma única operação `apply`, incluindo a manutenção
   dos índices e do log. Fontes originais não participam da lista de arquivos graváveis.
5. Retornar um recibo dos conceitos criados/alterados e das pendências preservadas.

Não presumir que cada fonte corresponda a uma única página ou que um resumo
substitua a integração com conceitos existentes. Em fontes conflitantes, manter
ambas as proveniências e a divergência explícita; o agente não fabrica consenso.

### Respostas que se tornam memória

`context` continua retornando trechos originais. O agente produz a resposta e
pode persistir comparações, decisões e conexões reutilizáveis como conceitos,
ligados às fontes efetivamente usadas. Isso é um passo de escrita explícito do
workflow, sem guardar automaticamente cada pergunta e resposta no log.

Uma síntese pode declarar a extensão `basis_revisions`, uma lista de objetos
`{path, revision}` para os conceitos locais dos quais depende. Mudança em uma
revisão torna a síntese candidata a revisão. O núcleo informa
`basis_outdated: true`, sem concluir que o texto é falso e sem reescrevê-lo.
Esse sinal é distinto de `stale_after` e não é um campo padronizado pelo OKF.
Mover um conceito atualiza referências conhecidas e esses caminhos no mesmo plano.

### Manutenção estrutural e revisão semântica

`validate --profile okf` verifica formato. `validate --profile health` acrescenta
checagens determinísticas: links quebrados, órfãos, páginas ausentes dos índices,
cobertura de fontes, sínteses com bases alteradas e validade temporal. Tipos que
dispensam fonte externa podem ser configurados; falta de fonte não prova falsidade.
Avisos de saúde não bloqueiam escrita apenas por existirem.

Contradições, lacunas conceituais e perguntas de pesquisa são analisadas pelo
agente com o contexto e as relações expostos. O núcleo não declara que BM25,
contagem de links ou um LLM infalível comprovam consistência semântica. Uma revisão
gera um relatório com evidências e sugestões; aplicar correções é outra operação.
Não iniciar busca web, agenda de revisão ou edição apenas porque um diagnóstico
apareceu. As instruções do workdir e a skill documentam os gatilhos de manutenção autorizados.

## 6. Interface CLI

Comandos de leitura oferecem `--json`. Em JSON, stdout contém exatamente um
documento JSON e nenhuma mensagem de progresso, ANSI ou log; stderr fica para
diagnóstico operacional. Em MCP stdio, stdout contém apenas mensagens do protocolo.

| Comando proposto | Finalidade |
| --- | --- |
| `kb status` | Escopo, integridade, contagens, arquivos ignorados e frescor do índice. |
| `kb list --para resources --limit 20` | Navegação paginada sem despejar corpos. |
| `kb search QUERY --limit 10` | Busca com snippets, linhas e metadados. |
| `kb read PATH --section HEADING` | Nota inteira ou seção com `revision`. |
| `kb context QUERY --max-tokens 2000` | Contexto limitado, com fontes e relações. |
| `kb links PATH --direction both` | Relações resolvidas de entrada e saída, para identificar notas afetadas. |
| `kb create --input request.json` | Criar nota com caminho ou categoria e título. |
| `kb patch PATH --input request.json` | Editar corpo e/ou metadados com versão esperada. |
| `kb move PATH NEW_PATH --input request.json` | Mover, verificar conflitos e atualizar links conhecidos. |
| `kb archive PATH --input request.json` | Atalho de move para a categoria Archive. |
| `kb apply --input plan.json --dry-run` | Validar e planejar um conjunto de criações, patches e movimentos; sem dry-run, aplicar uma transação. |
| `kb validate --profile okf` | Diagnósticos normativos e políticas separadas. |
| `kb index rebuild` | Reconstruir cache de pesquisa sem alterar notas. |
| `kb maintain --dry-run` | Mostrar reparos de índices, links e log; aplicar somente com `--apply`. |
| `kb export DIR` | Exportar snapshot validado do escopo, com manifesto de inclusões/exclusões. |
| `kb mcp` | Servidor stdio; `--read-only` remove e rejeita ferramentas de escrita. |

`create`, `patch`, `move`, `archive`, `apply` e `maintain` aceitam `--dry-run`. O arquivo
de entrada pode ser `-` para stdin. Não interpolar conteúdo Markdown em comandos
de shell. Exportação pode ficar no último incremento do MVP, após a integridade
das operações básicas.

Envelope de CLI, compartilhado pelo núcleo:

```json
{
  "schema_version": "kb.v1",
  "ok": true,
  "data": {},
  "warnings": [],
  "meta": {"root": "/base", "generation": "sha256:...", "index_stale": false}
}
```

Falha usa `ok: false`, `error: {code, message, details, retryable}` e não devolve
conteúdo de segredos ou arquivos externos. Códigos de saída: `0` sucesso, `1`
validação reprovada, `2` uso/configuração, `3` não encontrado, `4` conflito de versão,
`5` acesso negado, `6` I/O/recuperação. Códigos de erro do domínio são estáveis
(`REVISION_CONFLICT`, `INVALID_FRONTMATTER`, `PATH_OUTSIDE_ROOT`, etc.).

## 7. Busca e leitura

MVP: índice lexical por nota e seção com BM25 ou equivalente documentado. Título,
caminho, descrição e tags recebem peso maior que o corpo. Busca em português deve
aceitar `memoria` e `memória`, preservando o texto original nos resultados.
Não fazer chamadas a embedding ou LLM. Tokenização de identificadores e termos
como `s3`, `k8s` e `crossplane` deve ser coberta por casos de teste.

Filtros: `para`, prefixo de caminho, `type`, tags, `status`, confiança derivada,
`stale` e presença de fontes. Valores na mesma lista são OR; campos distintos
são AND. Incluir arquivo e deprecated é opt-in, com indicadores de que foram
excluídos. Notas expiradas continuam disponíveis e vêm marcadas, sem se tornarem
automaticamente deprecated. Busca literal e consulta vazia para inventário têm
semânticas distintas e documentadas; operadores de regex não são implícitos.

Cada hit contém `id`, `path`, `title`, `type`, `para`, `revision`, `score`,
`parse_status`, `okf`, e snippets `{heading, line_start, line_end, text}`.
Ordem determinística para empates por caminho. Paginação por cursor vinculado à
consulta e geração: índice alterado produz aviso/reinício explícito.

`read` retorna Markdown, frontmatter, revisão, faixas de linhas e indicadores
OKF. Leitura por seção deve tratar headings repetidos sem escolher um deles
silenciosamente: devolver identificadores de seção ou erro de ambiguidade.

`context` empacota trechos originais com fonte e linhas, não respostas geradas.
Respeita limite global de bytes e orçamento de tokens estimados; identifica o
estimador, truncamentos e itens omitidos. Relações opcionais são limitadas a um
salto. Não encher contexto com índices e logs só porque contêm muitas palavras-chave.

Atualizações feitas fora da ferramenta precisam aparecer após uma verificação
de frescor. Sem daemon obrigatório: cache incremental usa inventário de arquivos,
mtime/tamanho e hashes. A leitura de uma nota e toda escrita conferem seu hash
atual; um cache atrasado nunca autoriza sobrescrever uma revisão diferente.

`links` retorna arestas de entrada/saída com origem (Markdown ou proveniência),
destino, localização e estado de resolução. Não infere relações semânticas pelo
nome dos arquivos. `read` pode incluir essa vizinhança limitada, permitindo ao
MCP atender a mesma operação sem um tool separado.

## 8. Escrita, concorrência e recuperação

Criação exige destino inexistente. `patch` e `move` exigem `expected_revision`.
Ausência ou divergência não altera arquivos. O agente deve reler e reaplicar;
nenhum retry cego de uma substituição total.

Exemplo de patch, também aceito pelo MCP:

```json
{
  "expected_revision": "sha256:...",
  "idempotency_key": "session-123/decision-redis-ttl",
  "summary": "Corrigir TTL após teste de expiração",
  "actor": "process:agent-session",
  "body_edits": [{"old_text": "TTL de 300 segundos", "new_text": "TTL de 600 segundos"}],
  "metadata_patch": {"status": "draft"}
}
```

Cada `old_text` deve ocorrer exatamente uma vez; zero ou múltiplas ocorrências
falham. Para substituição de seção ou corpo completo, revisão esperada continua
obrigatória. Remoção de metadados usa operação explícita `remove_metadata_keys`;
não atribuir significado destrutivo implícito a `null`.

### Operações coordenadas em várias notas

`apply` recebe `operations`, `idempotency_key`, `summary` e `actor`. Cada item
usa a mesma gramática e as precondições de `create`, `patch` ou `move`, com um
`operation_id` único. Limite inicial: 50 operações e 4 MiB por plano. `--dry-run`
retorna arquivos afetados, diffs limitados, diagnósticos e revisões, sem gravar.

Aplicar usa todo o plano em uma transação lógica: uma revisão em conflito, um
destino ocupado ou uma nota inválida cancela o conjunto antes da publicação.
Revalidar também as dependências de sínteses informadas no plano. Índices e log
são derivados da transação, não operações que o agente precisa enumerar. Recibos
distinguem criações, alterações, movimentos e arquivos mantidos sem mudanças.

Referências entre notas criadas no mesmo plano devem ser resolvidas no estado
final proposto. Múltiplas operações sobre o mesmo arquivo ou destinos sobrepostos
são recusados no MVP; o agente deve consolidar os patches. Mover e editar o
mesmo conceito exige chamadas separadas no MVP, com revisão atualizada entre elas.
Os mesmos limites de isolamento e recuperação
abaixo valem para apply; não prometer transação atômica a leitores externos.

Uma operação lógica inclui nota, mudanças de links, regiões de índice e entrada
de log. O núcleo adquire lock por base, revalida revisões, planeja o diff e valida
os arquivos afetados antes de gravar. Registra journal durável em `.kb/state/`, grava
temporários no mesmo filesystem, sincroniza e substitui por rename. Após crash,
a próxima invocação recupera ou reverte antes de atender operações. Cache só
publica a geração nova após sucesso. Falha deve informar se algum estado precisa
de recuperação; nunca responder sucesso a uma aplicação parcial.

Não há rename atômico de vários arquivos: leitores externos podem observar um
estado transitório. Clientes da ferramenta usam o lock/geração. Editores externos
que não respeitam o lock continuam sujeitos a corridas; detectar mudanças até o
último ponto possível e documentar esse limite, sem prometer isolamento absoluto.

`idempotency_key` mais hash do pedido identifica um efeito. Repetição do mesmo
pedido retorna o resultado anterior sem duplicar nota ou log; reutilizar a chave
com outro pedido falha. Guardar recibos no journal e reconstruí-los na recuperação,
para que um crash entre publicação e resposta não duplique efeitos.

Sem commit, pull, push, hook Git ou mudança de branch implícitos. Git é uma camada
opcional operada separadamente. MVP não expõe exclusão permanente: arquivamento
e movimentação atendem o ciclo de memória.

## 9. Proveniência, confiança e validade

- `sources` é lista de objetos com `resource`; preservar campos desconhecidos.
  Referências externas são inertes e não são baixadas automaticamente.
- `generated.by` identifica o ator real; `generated.at` recebe instante com fuso
  em uma alteração significativa. Não reescrever a data em um no-op.
- `verified` aceita lista ou mapping único. Sem ele: unverified; apenas atores
  não humanos: machine-confirmed; presença de `human:`: human-reviewed.
- Escrita por agente não pode criar uma alegação de revisão humana. Não deduzir
  identidade humana de um token ou parâmetro de ator fornecido pelo modelo.
- Mudança substantiva no corpo ou nas afirmações remove `verified` ativo e deixa
  o motivo no log. Uma ferramenta de revisão humana poderá ser adicionada depois,
  com confirmação fora do controle do modelo. Esses controles são política do
  produto, além dos requisitos mínimos do OKF.
- `stale_after` segue o instante ISO 8601 com fuso da revisão oficial fixada nesta
  spec. Datas sem hora recebem diagnóstico e não devem ser
  silenciosamente tratadas como meia-noite em um fuso adivinhado. `--as-of` permite
  testes determinísticos. Ausência significa validade não especificada.
- Attested Computation é lido e validado estaticamente; não executar código,
  acessar executores ou declarar que um recibo foi atestado.

## 10. Índices e log

Pesquisa usa o inventário em disco, não depende de `index.md` estar completo.
Índices atendem a navegação progressiva: arquivos diretos e links para índices
das subpastas, com descrição quando disponível.

Para não destruir os índices manuais atuais, a manutenção usa regiões geradas
delimitadas por comentários HTML. Preserva frontmatter permitido, introdução e
links manuais. Adotar um índice inteiro como gerado exige `maintain --dry-run`
e ação explícita de takeover. Detectar duplicatas entre conteúdo manual e região
gerada. Manter uma ordem determinística e escapar Markdown vindo de títulos.

O log global registra uma linha concisa por operação lógica sob a data do fuso
configurado, mais recente primeiro, com links e resumo. Instantes de metadados
continuam com fuso explícito. Se um arquivo reservado existente for inválido ou
um reparo exigir apagar texto manual, a escrita falha antes de modificar notas,
com proposta de reparo. Nunca substituir todo o log existente.

O tipo de evento (`ingest`, `synthesis`, `maintenance` ou `edit`) pode ser gravado
na linha para facilitar a leitura. Preservar eventos anteriores; uma correção é
um novo evento. Continuar usando seções de data `YYYY-MM-DD` do OKF, mesmo que o
exemplo do gist use outro formato. Não gerar evento para toda consulta sem efeito.

## 11. Interface MCP

Mesmo núcleo e DTOs da CLI. Negociar a versão suportada pelo SDK oficial; usar
MCP `2025-11-25` como baseline de interoperabilidade comprovada nos experimentos.
Não implementar uma variante ad hoc de JSON-RPC. Transportes futuros devem
reutilizar o núcleo e declarar seus requisitos de autenticação separadamente.

| Tool | Entrada principal | Resultado |
| --- | --- | --- |
| `kb_status` | sem argumentos | Escopo, índice e capacidades. |
| `kb_list` | pasta/categoria, cursor, limite | Inventário paginado. |
| `kb_search` | query, filtros, cursor, limite | Hits estruturados. |
| `kb_read` | path, seção/faixa e relações opcionais | Fonte, revisão e vizinhança limitada. |
| `kb_context` | query, filtros, orçamento | Trechos citáveis limitados. |
| `kb_create` | path ou categoria+título, type, body, metadata, resumo, chave | Recibo e revisão. |
| `kb_patch` | path, expected_revision, edições, resumo, chave | Diff resumido e revisão nova. |
| `kb_move` | path, destino, expected_revision, resumo, chave | Mapeamento e links alterados. |
| `kb_apply` | operations, resumo, chave, ator, dry_run | Plano ou recibo de aplicação coordenada. |
| `kb_validate` | escopo e perfil okf/health | Relatório com camadas distintas. |

Schemas de entrada fechados, descrições pequenas, limites explícitos. Não
oferecer shell genérico ou dezenas de tools de UI. Em modo read-only, ferramentas
de escrita somem de `tools/list` e continuam bloqueadas em chamadas diretas.

Retornar `structuredContent` com envelope de domínio e conteúdo textual compacto
para clientes compatíveis. Falha de ferramenta usa `isError: true` mais código
de domínio, em vez de esconder `ok: false` dentro de um sucesso de protocolo.
Erros de validação da base são dados do relatório, não necessariamente falha de
execução do tool. Marcar `readOnlyHint` e demais annotations corretamente, sem
tratá-las como controle de acesso.

Expor recursos paginados em URIs `kb://local/...` com caminhos percent-encoded e
revisão no resultado. Essa URI é extensão de transporte, não o Concept ID OKF.
O servidor fica preso à raiz de inicialização: um argumento do modelo não troca
a base para outro diretório. Prompts e subscriptions não são requisitos do MVP.

## 12. Limites operacionais

Recusar caminhos fora da raiz, `..`, caminhos absolutos em operações de arquivos,
symlinks e escapes por normalização. Links de conhecimento fora do bundle podem
ser exibidos como referências, mas não abertos pela camada de arquivos. Tratar
colisões de caixa em filesystems e destinos já existentes antes de mover.

Valores iniciais: nota de até 1 MiB; leitura de até 256 KiB por resposta; consulta
de até 4.096 caracteres; 10 hits padrão, máximo 100; contexto de 2.000 tokens
estimados padrão, máximo 16.000 e teto de 256 KiB; limites também para profundidade
YAML e número de operações de patch. Limites configuráveis têm teto documentado.

Sem telemetria nem acesso à rede por padrão. Erros não incluem ambiente inteiro,
credenciais ou arquivos fora do escopo. Cache é local, gravado com permissões
adequadas ao usuário, e não altera a política de acesso dos arquivos originais.

## 13. Arquitetura e linguagem

Camadas: filesystem/escopo → parser e modelo → inventário/busca/relações → serviço
de leitura e transações → adaptadores CLI e MCP. Nenhuma regra de negócio mora
apenas em um adaptador. Modelo e parser não dependem de um servidor MCP em execução.

Baseline adotada para a implementação: **Go**, pela distribuição em binário e pela existência de
referências próximas em Go. Rust também atende, especialmente se o protótipo
encontrar vantagens práticas de parsing e indexação. Os testes realizados não
comparam implementações próprias em Rust e Go e não provam superioridade de uma.

Escolher bibliotecas e versões no protótipo, contra documentação atual. Usar SDK
MCP oficial. Índice lexical pode usar SQLite FTS5 ou uma biblioteca local de BM25;
é cache e precisa passar o mesmo conjunto de consultas. Se SQLite exigir CGO ou
dependências de plataforma, avaliar o custo real antes de fixar a escolha.
Não embutir modelos de dezenas de megabytes no binário básico.

Para a implementação em Go, o [levantamento spf13](research.md)
registra Cobra como candidato preferencial para o adaptador CLI, com pflag pelo
ecossistema dele. Viper, Afero, Fileflow e Pathologize têm aplicações condicionais
e limites frente ao contrato desta spec. As recomendações são insumos para o
protótipo; não fixam dependências nem substituem os requisitos de revisão,
durabilidade, confinamento e preservação de conteúdo. Os guias go-skills e
cli-style servem como referências de desenvolvimento e revisão, sujeitos às
escolhas explícitas do projeto.

## 14. Critérios de aceite e plano de entrega

| ID | Evidência exigida |
| --- | --- |
| A01 | Busca/read funcionam sem UI, Git, modelo, credencial ou rede. |
| A02 | CLI e MCP retornam os mesmos IDs, revisões, filtros e erros em fixtures equivalentes. |
| A03 | Concept mínimo com só `type` é conformante; tipos e campos desconhecidos sobrevivem. |
| A04 | Notas sem YAML ou type são reportadas como inválidas; escrita normal não aceita uma base não conformante. Não há requisito de busca ou migração de legado. |
| A05 | YAML inválido não oculta silenciosamente notas; status informa quantos arquivos não foram parseados. |
| A06 | Mapping único em `verified` é aceito; mudança de conteúdo invalida a revisão ativa. |
| A07 | Instante `stale_after` com fuso é interpretado corretamente; formato incompatível é diagnosticado. |
| A08 | Busca com e sem acentos recupera o mesmo conjunto esperado em português. |
| A09 | Editar corpo preserva comentário YAML e campos não tocados; no-op preserva bytes e não gera log. |
| A10 | Duas escritas com a mesma revisão: apenas a primeira aplica; conflito não muda nota, índice ou log. |
| A11 | Retry com mesma chave não duplica efeitos, inclusive após falha entre publicação e resposta. |
| A12 | Create/edit atualizam nota, índice e log; falhas injetadas em cada etapa permitem recuperação. |
| A13 | Move atualiza links Markdown resolvidos e âncoras, sem alterar referências ambíguas ou externas indevidamente. |
| A14 | Traversal, symlinks, colisões e arquivo fora da raiz não permitem leitura nem escrita. |
| A15 | `--read-only` oculta e recusa mutações; stdout MCP não contém logs. |
| A16 | Contexto respeita teto global, informa truncamentos e fornece linhas/revisões das fontes. |
| A17 | Editar via editor externo invalida cache e revisão na próxima operação relevante. |
| A18 | Exportar inclui relatório e lista de exclusões; não anuncia conformidade de arquivos não examinados. |
| A19 | Instalação/init não alteram configs globais ou registram agentes automaticamente. |
| A20 | Trocar o perfil de navegação PARA por outra hierarquia não altera parsing, identidade e validade OKF dos mesmos arquivos. |
| A21 | Uma ingestão sintética cria uma nota de fonte e atualiza duas notas existentes em apply; conflito em uma delas não publica nenhuma mudança. |
| A22 | Os hashes de snapshots originais permanecem iguais após criar, editar, mover e aplicar planos na wiki. |
| A23 | Uma síntese com basis_revisions de dois conceitos é marcada para revisão quando um deles muda; não é reescrita nem declarada falsa automaticamente. |
| A24 | Health identifica órfão, link ausente, conceito fora do índice e dependência alterada, sem alegar detecção determinística de contradições. |
| A25 | Comparação persistida pelo agente mantém proveniência; repetir uma consulta sem aprendizado não cria nota ou evento. |
| A26 | Sem AGENTS.md dentro da base, CLI/MCP operam normalmente. No agente alvo, instrução do workdir e skill instalada conduzem consulta à base externa antes de responder; aprendizado reutilizável é atualizado e consulta sem novidade não escreve. Registrar evidência observada, sem prometer adesão universal dos modelos. |

Incrementos: (1) parser/escopo/validate/read; (2) índice/search/context e consultas
de referência; (3) criação/patch/apply/transações/log/índice; (4) move/archive,
links e health; (5) MCP, workflows de ingestão/síntese/revisão, exportação e
distribuição. Validar cada incremento antes do próximo. Workflows são instruções
para o agente externo, sem chamadas a LLM adicionadas ao núcleo.

Medir em macOS arm64 e Linux amd64/arm64: tamanho de binário, RSS, início frio,
reindexação e p50/p95 de busca, declarando hardware e corpus. Metas iniciais para
10 mil notas sintéticas de 2 KiB: busca aquecida p95 ≤ 150 ms e RSS ≤ 150 MiB;
reindexação completa ≤ 10 s. São metas a validar, não resultados já obtidos.

Decisões pendentes para a implementação: biblioteca de
busca e política de retenção dos recibos de idempotência. A interface e os testes
acima permitem avaliar essas escolhas sem reescrever o contrato dos agentes.

### Revisão de prontidão — 2026-10-24

A v0.3 está pronta como direção de produto e base de um protótipo técnico. Não
é ainda um contrato integralmente executável. Antes de implementar as operações
duráveis e publicar uma interface estável, fechar:

1. Schemas versionados de entrada e saída de cada operação, erros e exemplos
   completos de planos. As tabelas e o envelope desta versão não substituem esses
   arquivos; a gramática de create/patch/move precisa ser inequívoca.
2. Máquina de estados do journal, ponto de commit, estratégia de recuperação,
   retenção de recibos e falhas injetadas. Resolver explicitamente o caso de uma
   leitura encontrar recuperação pendente: a regra de não modificar notas em
   consultas deve ser conciliada com a recuperação descrita na seção 8.
3. Contrato de reparo de base inválida: comando/entrada, mudanças permitidas e
   validação final. Também fechar a descoberta de raiz no primeiro init e o
   contrato do setup explícito no workdir. A instrução de entrada fica no workdir;
   AGENTS.md dentro do bundle é opcional, conforme a seção 3.
4. Validar no protótipo parser com preservação de YAML, índice lexical e SDK MCP;
   registrar versões e limitações em decisões de arquitetura. Go é a baseline
   adotada no bootstrap do repositório; parser e busca ainda serão prototipados.

O nome final não bloqueia o protótipo. As metas de desempenho continuam sem
medição. Ver [desenho do projeto](project-overview.md) para a explicação do produto,
arquitetura e sequência de entrega. Esta revisão esclarece a v0.3, sem declarar
implementação, validação de desempenho ou aprovação das escolhas pendentes.

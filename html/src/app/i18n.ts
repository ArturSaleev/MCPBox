export type Language = 'en' | 'ru';

export type Dictionary = {
  labels: {
    appTitle: string;
    controlCenter: string;
    appDescription: string;
    projects: string;
    createProject: string;
    duplicateProject: string;
    name: string;
    description: string;
    prompt: string;
    save: string;
    saving: string;
    projectOverview: string;
    servers: string;
    running: string;
    connectionEndpoint: string;
    addServer: string;
    serverName: string;
    launchCommand: string;
    stdio: string;
    httpStreaming: string;
    workingDirectory: string;
    url: string;
    autoStart: string;
    copied: string;
    copyUrl: string;
    noProjectSelected: string;
    ready: string;
    notSelected: string;
    notSpecified: string;
    stop: string;
    start: string;
    language: string;
    english: string;
    russian: string;
    arguments: string;
    environmentVariables: string;
    environmentVariablePassthrough: string;
    bearerTokenEnvironmentVariable: string;
    headers: string;
    headersFromEnvironmentVariables: string;
    addArgument: string;
    addEnvironmentVariable: string;
    addHeader: string;
    addVariable: string;
    key: string;
    value: string;
    logs: string;
    auditLogs: string;
    performance: string;
    timeWindow: string;
    refresh: string;
    launchProject: string;
    launchOllama: string;
    launchLlamaCpp: string;
    launchLMStudio: string;
    ollamaModel: string;
    llamaCppModel: string;
    llamaCppModelPath: string;
    llamaCppModelName: string;
    pauseProject: string;
    resumeProject: string;
    paused: string;
    disabled: string;
    disableServer: string;
    enableServer: string;
    remote: string;
    unknownActor: string;
    requestControl: string;
    filterByProject: string;
    last5Minutes: string;
    lastHour: string;
    last24Hours: string;
    allProjects: string;
    currentProjectOnly: string;
    activityOverview: string;
    hottestProject: string;
    hottestServer: string;
    requests: string;
    errors: string;
    avgLatency: string;
    p95Latency: string;
    trafficIn: string;
    trafficOut: string;
    errorRate: string;
    noActivity: string;
    consoleFeed: string;
    requestVolume: string;
    latencyTrend: string;
    topSlowServers: string;
    topErrorServers: string;
    topTrafficServers: string;
    recentFailures: string;
    info: string;
    serverInfo: string;
    capabilities: string;
    tools: string;
    resources: string;
    prompts: string;
    readme: string;
    instructions: string;
    protocolVersion: string;
    version: string;
    noReadme: string;
    noTools: string;
    noResources: string;
    noPrompts: string;
    inspectServer: string;
    health: string;
    healthy: string;
    failed: string;
    unknown: string;
    lastCheck: string;
    check: string;
    oauth: string;
    connected: string;
    notConnected: string;
    oauthConnected: string;
    market: string;
    integrations: string;
    catalog: string;
    catalogItems: string;
    installed: string;
    lastSync: string;
    externalManifestUrl: string;
    manifestSource: string;
    serverSource: string;
    localFileSource: string;
    chooseFile: string;
    syncCatalog: string;
    allCategories: string;
    command: string;
    endpoint: string;
    runtime: string;
    source: string;
    installModel: string;
    systemDependencies: string;
    docs: string;
    website: string;
    generalCategory: string;
    mcpDiscovery: string;
    installPackage: string;
    uninstallPackage: string;
    packageInstalled: string;
    packageNotInstalled: string;
    addToProject: string;
    installFirst: string;
    addedToProject: string;
    notInProject: string;
    knowledgeBase: string;
    collections: string;
    create: string;
    actions: string;
    edit: string;
    createCollection: string;
    connect: string;
    delete: string;
    disconnect: string;
    index: string;
    search: string;
    connectedKnowledgeBases: string;
    toolContract: string;
    mcpToolReady: string;
    showMore: string;
    hide: string;
    manageTools: string;
    enabled: string;
    bearerEndpointProtection: string;
    bearerToken: string;
    generateToken: string;
    showToken: string;
    hideToken: string;
    copyToken: string;
  };
  messages: {
    loadingProjects: string;
    noProjects: string;
    projectHelper: string;
    projectNamePlaceholder: string;
    projectDescriptionPlaceholder: string;
    emptySelection: string;
    emptySelectionBody: string;
    overviewFallbackDescription: string;
    connectionDescription: string;
    connectionWarning: (token: string) => string;
    addServerDescription: string;
    serverNamePlaceholder: string;
    launchCommandPlaceholder: string;
    commandPlaceholder: string;
    argumentPlaceholder: string;
    workingDirectoryPlaceholder: string;
    urlPlaceholder: string;
    bearerTokenPlaceholder: string;
    envPassthroughPlaceholder: string;
    serverControlDescription: string;
    noServers: string;
    workspaceGroupFallback: string;
    serverCount: (count: number) => string;
    runningCount: (count: number) => string;
    createProjectError: string;
    duplicateProjectError: string;
    loadProjectsError: string;
    addServerError: string;
    startServerError: string;
    stopServerError: string;
    setProjectPausedError: string;
    setServerEnabledError: string;
    logsDescription: string;
    performanceDescription: string;
    loadingLogs: string;
    loadingMetrics: string;
    noLogs: string;
    noMetrics: string;
    projectTag: (id: number) => string;
    serverTag: (id: number) => string;
    consoleDescription: string;
    popularityDescription: string;
    inspectDescription: string;
    inspectServerError: string;
    checkServerError: string;
    checkServerHealthy: (name: string) => string;
    checkServerFailed: (name: string, reason: string) => string;
    requestFailed: (status: number) => string;
    marketDescription: string;
    searchCatalogPlaceholder: string;
    catalogResultsSummary: (visible: number, total: number) => string;
    notSynced: string;
    advancedModeEnabled: string;
    localCatalogFileSelected: (name: string) => string;
    localCatalogFileMissing: string;
    selectProjectBeforeInstall: string;
    noDescriptionProvided: string;
    upstreamAuthNotice: string;
    noValue: string;
    workingDirectoryValue: (path: string) => string;
    autoStartAfterInstall: string;
    syncManifestToPopulateCatalog: string;
    noIntegrationsInCategory: string;
    installIntegrationTitle: (name: string) => string;
    installIntegrationFallbackTitle: string;
    installIntegrationDescription: string;
    oneValuePerLine: string;
    installIntegrationAction: string;
    loadPackagesError: string;
    installPackageError: string;
    addPackageToProjectError: string;
    uninstallPackageError: string;
    catalogInstallAdded: (name: string) => string;
    catalogHealthCheckPassed: (name: string) => string;
    catalogHealthCheckFailed: (name: string) => string;
    catalogHealthCheckFailedWithReason: (name: string, reason: string) => string;
    addPackageDialogTitle: (name: string) => string;
    addPackageDialogFallbackTitle: string;
    addPackageDialogDescription: string;
    sharedInstallMode: string;
    projectInstallMode: string;
    multiProjectSupported: string;
    singleProjectOnly: string;
    packageUsageCount: (count: number) => string;
    packageInUseCannotUninstall: string;
    systemDependencyVersion: (name: string, version: string) => string;
    systemDependencyRequired: (name: string) => string;
    envSchemaDescription: string;
    launchOllamaError: string;
    launchLlamaCppError: string;
    launchLMStudioError: string;
    noOllamaModels: string;
    knowledgeBaseHeroTitle: string;
    knowledgeBaseHeroDescription: string;
    createKnowledgeBaseTitle: string;
    createKnowledgeBaseDescription: string;
    editKnowledgeBaseDescription: string;
    collectionIdLabel: string;
    collectionIdPlaceholder: string;
    collectionNamePlaceholder: string;
    indexPathLabel: string;
    indexPathPlaceholder: string;
    sourceFolderTitle: string;
    sourceFolderPlaceholder: string;
    autoReindexTitle: string;
    autoReindexDescription: string;
    autoReindexBadge: string;
    noKnowledgeBasesCreated: string;
    indexFolderTitle: string;
    indexFolderDescription: string;
    indexFolderPlaceholder: string;
    supportedFormatsLabel: string;
    supportedFormatsValue: string;
    searchCollectionTitle: string;
    searchCollectionDescription: string;
    searchCollectionPlaceholder: string;
    searchResultsTitle: string;
    searchResultsDescription: (name: string) => string;
    searchResultsEmpty: string;
    searchQueryRequired: string;
    deleteKnowledgeBaseConfirm: string;
    connectedKnowledgeBasesDescription: string;
    connectKnowledgeBaseTitle: string;
    connectKnowledgeBaseDescription: string;
    noAvailableCollections: string;
    noKnowledgeBasesConnected: string;
    mcpToolReadyIntro: string;
    mcpToolReadyOutro: string;
    otherConnectionOptions: string;
    otherConnectionOptionsDescription: string;
    duplicateProjectDescription: string;
    duplicateProjectNamePlaceholder: string;
    manageToolsDescription: string;
    noServerTools: string;
    loadServerToolsError: string;
    updateServerToolsError: string;
    disabledToolsBadge: (count: number) => string;
    launchProjectDescription: string;
    ollamaNotInstalled: string;
    llamaCppNotInstalled: string;
    llamaCppNotConfigured: string;
    llamaCppFilePickerHint: string;
    bearerEndpointProtectionDescription: string;
    bearerTokenGeneratedAfterCreate: string;
    bearerTokenRegenerated: string;
  };
};

export const defaultLanguage: Language = 'en';
export const languageStorageKey = 'mcpbox.language';

export const dictionaries: Record<Language, Dictionary> = {
  en: {
    labels: {
      appTitle: 'MCPBox',
      controlCenter: '',
      appDescription: 'Unified service for managing MCP servers.',
      projects: 'Projects',
      createProject: 'Create Project',
      duplicateProject: 'Duplicate project',
      name: 'Name',
      description: 'Description',
      prompt: 'Prompt',
      save: 'Save',
      saving: 'Saving...',
      projectOverview: 'Project Overview',
      servers: 'Servers',
      running: 'Running',
      connectionEndpoint: 'Connection Endpoint',
      addServer: 'Add MCP Server',
      serverName: 'Server name',
      launchCommand: 'Launch command',
      stdio: 'STDIO',
      httpStreaming: 'HTTP streaming',
      workingDirectory: 'Working directory',
      url: 'URL',
      autoStart: 'Start automatically when MCPBox boots',
      copied: 'Copied',
      copyUrl: 'Copy URL',
      noProjectSelected: 'No project selected',
      ready: 'Ready',
      notSelected: 'Not selected',
      notSpecified: 'Not specified',
      stop: 'Stop',
      start: 'Start',
      language: 'Language',
      english: 'English',
      russian: 'Russian',
      arguments: 'Arguments',
      environmentVariables: 'Environment variables',
      environmentVariablePassthrough: 'Environment variable passthrough',
      bearerTokenEnvironmentVariable: 'Bearer token environment variable',
      headers: 'Headers',
      headersFromEnvironmentVariables: 'Headers from environment variables',
      addArgument: 'Add argument',
      addEnvironmentVariable: 'Add environment variable',
      addHeader: 'Add header',
      addVariable: 'Add variable',
      key: 'Key',
      value: 'Value',
      logs: 'Logs',
      auditLogs: 'Audit Logs',
      performance: 'Performance',
      timeWindow: 'Time window',
      refresh: 'Refresh',
      launchProject: 'Launch',
      launchOllama: 'Launch Ollama',
      launchLlamaCpp: 'Launch llama.cpp',
      launchLMStudio: 'Add to LM Studio',
      ollamaModel: 'Ollama model',
      llamaCppModel: 'llama.cpp model',
      llamaCppModelPath: 'Model path',
      llamaCppModelName: 'Model name',
      pauseProject: 'Pause project',
      resumeProject: 'Resume project',
      paused: 'Paused',
      disabled: 'Disabled',
      disableServer: 'Disable server',
      enableServer: 'Enable server',
      remote: 'Remote',
      unknownActor: 'unknown actor',
      requestControl: 'Request control',
      filterByProject: 'Filter by project',
      last5Minutes: 'Last 5 minutes',
      lastHour: 'Last hour',
      last24Hours: 'Last 24 hours',
      allProjects: 'All projects',
      currentProjectOnly: 'Current project only',
      activityOverview: 'Activity overview',
      hottestProject: 'Hottest project',
      hottestServer: 'Hottest server',
      requests: 'requests',
      errors: 'Errors',
      avgLatency: 'Avg latency',
      p95Latency: 'P95 latency',
      trafficIn: 'Traffic in',
      trafficOut: 'Traffic out',
      errorRate: 'Error rate',
      noActivity: 'No activity yet',
      consoleFeed: 'Console feed',
      requestVolume: 'Request volume',
      latencyTrend: 'Latency trend',
      topSlowServers: 'Top slow servers',
      topErrorServers: 'Top error servers',
      topTrafficServers: 'Top traffic servers',
      recentFailures: 'Recent failures',
      info: 'Info',
      serverInfo: 'Server Info',
      capabilities: 'Capabilities',
      tools: 'Tools',
      resources: 'Resources',
      prompts: 'Prompts',
      readme: 'README',
      instructions: 'Instructions',
      protocolVersion: 'Protocol version',
      version: 'Version',
      noReadme: 'README not found near the local server path.',
      noTools: 'No tools exposed.',
      noResources: 'No resources exposed.',
      noPrompts: 'No prompts exposed.',
      inspectServer: 'Inspect server',
      health: 'Health',
      healthy: 'Healthy',
      failed: 'Failed',
      unknown: 'Unknown',
      lastCheck: 'Last check',
      check: 'Check',
      oauth: 'OAuth',
      connected: 'Connected',
      notConnected: 'Not connected',
      oauthConnected: 'OAuth connected',
      market: 'Market',
      integrations: 'Integrations',
      catalog: 'Catalog',
      catalogItems: 'Items',
      installed: 'Installed',
      lastSync: 'Last sync',
      externalManifestUrl: 'External manifest URL',
      manifestSource: 'Catalog source',
      serverSource: 'Server',
      localFileSource: 'Local file',
      chooseFile: 'Choose file',
      syncCatalog: 'Sync catalog',
      allCategories: 'All categories',
      command: 'Command',
      endpoint: 'Endpoint',
      runtime: 'Runtime',
      source: 'Source',
      installModel: 'Install model',
      systemDependencies: 'System dependencies',
      docs: 'Docs',
      website: 'Website',
      generalCategory: 'general',
      mcpDiscovery: 'mcp discovery',
      installPackage: 'Install package',
      uninstallPackage: 'Uninstall package',
      packageInstalled: 'Package installed',
      packageNotInstalled: 'Package not installed',
      addToProject: 'Add to project',
      installFirst: 'Install package first',
      addedToProject: 'Added to project',
      notInProject: 'Not in project',
      knowledgeBase: 'Knowledge Base',
      collections: 'Collections',
      create: 'Create',
      actions: 'Actions',
      edit: 'Edit',
      createCollection: 'Create Collection',
      connect: 'Connect',
      delete: 'Delete',
      disconnect: 'Disconnect',
      index: 'Index',
      search: 'Search',
      connectedKnowledgeBases: 'Connected Knowledge Bases',
      toolContract: 'Tool Contract',
      mcpToolReady: 'MCP Tool Ready',
      showMore: 'Show more',
      hide: 'Hide',
      manageTools: 'Tools',
      enabled: 'Enabled',
      bearerEndpointProtection: 'Bearer endpoint protection',
      bearerToken: 'Bearer token',
      generateToken: 'Generate token',
      showToken: 'Show token',
      hideToken: 'Hide token',
      copyToken: 'Copy token',
    },
    messages: {
      loadingProjects: 'Loading projects...',
      noProjects: 'No projects yet. Create the first workspace below.',
      projectHelper: 'A project is a logical group of MCP servers for one environment.',
      projectNamePlaceholder: 'Client Workspace',
      projectDescriptionPlaceholder: 'What this workspace group is for',
      emptySelection:
        'Create the first workspace on the left and it will become the control center for its MCP servers.',
      emptySelectionBody:
        'Create the first workspace on the left and it will become the control center for its MCP servers.',
      overviewFallbackDescription: 'Workspace group for MCP clients and tools.',
      connectionDescription:
        'Use the project endpoint with Authorization: Bearer <project-token>. Clients connected through this endpoint can access all enabled MCP servers in the project.',
      connectionWarning: (_token: string) => 'To use this endpoint, add and enable at least one MCP server or connect a knowledge base.',
      addServerDescription:
        'Add one or more MCP servers to expose them through the shared project endpoint.',
      serverNamePlaceholder: 'Filesystem Server',
      launchCommandPlaceholder: 'npx -y @modelcontextprotocol/server-filesystem /path',
      commandPlaceholder: 'uvx mcp-server or node dist/index.js',
      argumentPlaceholder: '--port or ./path',
      workingDirectoryPlaceholder: '/absolute/path or leave empty',
      urlPlaceholder: 'https://mcp.example.com/mcp',
      bearerTokenPlaceholder: 'MCP_BEARER_TOKEN',
      envPassthroughPlaceholder: 'OPENAI_API_KEY',
      serverControlDescription:
        'Manage the processes and availability of all MCP servers exposed through the project endpoint.',
      noServers: 'No servers have been added to this project yet.',
      workspaceGroupFallback: 'Workspace group for MCP clients',
      serverCount: (count: number) => `${count} servers`,
      runningCount: (count: number) => `${count} running`,
      createProjectError: 'Failed to create project',
      duplicateProjectError: 'Failed to duplicate project',
      loadProjectsError: 'Failed to load projects',
      addServerError: 'Failed to add server',
      startServerError: 'Failed to start server',
      stopServerError: 'Failed to stop server',
      setProjectPausedError: 'Failed to update project state',
      setServerEnabledError: 'Failed to update server state',
      logsDescription: 'Request monitoring and control actions across projects and MCP servers.',
      performanceDescription:
        'Latency, failures, and traffic across MCP servers without leaving the logs screen.',
      loadingLogs: 'Loading logs...',
      loadingMetrics: 'Loading metrics...',
      noLogs: 'No audit logs yet.',
      noMetrics:
        'No performance data yet. Metrics will appear after requests start flowing through MCPBox.',
      projectTag: (id: number) => `project #${id}`,
      serverTag: (id: number) => `server #${id}`,
      consoleDescription: 'Compact event stream for requests, connects, and control actions.',
      popularityDescription:
        'Who is receiving the most MCP traffic right now based on the current filter.',
      inspectDescription:
        'Live MCP inspection for this local STDIO server plus nearby README if found.',
      inspectServerError: 'Failed to inspect server',
      checkServerError: 'Failed to verify server health',
      checkServerHealthy: (name: string) => `${name} passed the health check.`,
      checkServerFailed: (name: string, reason: string) => `${name} failed the health check: ${reason}`,
      requestFailed: (status: number) => `Request failed with status ${status}`,
      marketDescription:
        'Sync the external integration manifest into SQLite and install selected items into the current project as linked MCP servers.',
      searchCatalogPlaceholder: 'Search integrations, tags, runtime, package',
      catalogResultsSummary: (visible: number, total: number) => `${visible} of ${total} integrations shown`,
      notSynced: 'Not synced',
      advancedModeEnabled: 'Advanced mode enabled. Press Cmd/Ctrl + Shift + U to hide.',
      localCatalogFileSelected: (name: string) => `Selected file: ${name}`,
      localCatalogFileMissing: 'Choose a local catalog JSON file before syncing.',
      selectProjectBeforeInstall:
        'Create or select a project before installing integrations.',
      noDescriptionProvided: 'No description provided.',
      upstreamAuthNotice:
        'Authentication is handled by the upstream MCP server. After install, your MCP client should complete the sign-in flow when it connects through MCPBox.',
      noValue: 'n/a',
      workingDirectoryValue: (path: string) => `Working directory: ${path}`,
      autoStartAfterInstall: 'Starts automatically after install',
      syncManifestToPopulateCatalog: 'Sync the external manifest to populate the catalog.',
      noIntegrationsInCategory: 'No integrations in this category yet.',
      installIntegrationTitle: (name: string) => `Install ${name}`,
      installIntegrationFallbackTitle: 'Install integration',
      installIntegrationDescription:
        'Fill in the required connection settings before adding this integration to the selected project.',
      oneValuePerLine: 'One value per line',
      installIntegrationAction: 'Install integration',
      loadPackagesError: 'Failed to load packages',
      installPackageError: 'Failed to install package',
      addPackageToProjectError: 'Failed to add package to project',
      uninstallPackageError: 'Failed to uninstall package',
      catalogInstallAdded: (name: string) => `${name} was added to the project.`,
      catalogHealthCheckPassed: (name: string) => `${name} was added and passed the health check.`,
      catalogHealthCheckFailed: (name: string) => `${name} was added, but the health check failed.`,
      catalogHealthCheckFailedWithReason: (name: string, reason: string) => `${name} was added, but the health check failed: ${reason}`,
      addPackageDialogTitle: (name: string) => `Add ${name} to project`,
      addPackageDialogFallbackTitle: 'Add package to project',
      addPackageDialogDescription:
        'Choose a project and configure the selected package instance before adding it.',
      sharedInstallMode: 'Shared package install',
      projectInstallMode: 'Project-scoped install',
      multiProjectSupported: 'Reusable across multiple projects',
      singleProjectOnly: 'Best for a single project',
      packageUsageCount: (count: number) => `Used in ${count} projects`,
      packageInUseCannotUninstall: 'This package is still connected to one or more projects.',
      systemDependencyVersion: (name: string, version: string) => `${name} ${version}+`,
      systemDependencyRequired: (name: string) => `${name} required`,
      envSchemaDescription: 'These values will be passed to the server process as environment variables.',
      launchOllamaError: 'Failed to launch Ollama terminal.',
      launchLlamaCppError: 'Failed to launch llama.cpp terminal.',
      launchLMStudioError: 'Failed to open LM Studio.',
      noOllamaModels: 'No local Ollama models found.',
      knowledgeBaseHeroTitle: 'Global Knowledge Collections',
      knowledgeBaseHeroDescription:
        'Create reusable collections once, index local folders, and then attach them to one or many projects. Supported formats include code, text, CSV, XLSX, DOCX, PPTX, and text-based PDF.',
      createKnowledgeBaseTitle: 'Create Knowledge Base',
      createKnowledgeBaseDescription:
        'Add a global collection, choose its source folder right away, and MCPBox will index it immediately. You can index code, text, spreadsheets, Office documents, and text-based PDFs.',
      editKnowledgeBaseDescription:
        'Update the collection name, source folder, or auto reindex setting. Saving will rebuild the index from the selected folder.',
      collectionIdLabel: 'Collection ID',
      collectionIdPlaceholder: 'crm_gym',
      collectionNamePlaceholder: 'CRM Gym Codebase',
      indexPathLabel: 'Index path',
      indexPathPlaceholder: '.mcpbox/rag/crm_gym.bleve',
      sourceFolderTitle: 'Source Folder',
      sourceFolderPlaceholder: '/Users/artur/projects/crm-gym',
      autoReindexTitle: 'Reindex every 10 minutes',
      autoReindexDescription:
        'When enabled, MCPBox will rebuild this collection automatically every 10 minutes while the app is running.',
      autoReindexBadge: 'Auto reindex: every 10 min',
      noKnowledgeBasesCreated: 'No knowledge bases created yet.',
      indexFolderTitle: 'Source Folder',
      indexFolderDescription:
        'Choose the local folder whose files should be added to this knowledge base. System folders like node_modules, vendor, build artifacts, and Python virtual environments are skipped automatically.',
      indexFolderPlaceholder: '/path/to/project',
      supportedFormatsLabel: 'Supported',
      supportedFormatsValue: 'Code, Text, CSV, XLSX, DOCX, PPTX, PDF (text-based)',
      searchCollectionTitle: 'Search Collection',
      searchCollectionDescription:
        'Run a quick keyword search and inspect the most relevant indexed chunks.',
      searchCollectionPlaceholder: 'payment gateway',
      searchResultsTitle: 'Search Results',
      searchResultsDescription: (name: string) => `Top matches from ${name}.`,
      searchResultsEmpty: 'No matching chunks were found for this query.',
      searchQueryRequired: 'Search query is required.',
      deleteKnowledgeBaseConfirm: 'Delete this knowledge base?',
      connectedKnowledgeBasesDescription:
        'Attach one or many global collections to this project.',
      connectKnowledgeBaseTitle: 'Connect Knowledge Base',
      connectKnowledgeBaseDescription:
        'Choose one of the global collections and attach it to this project.',
      noAvailableCollections:
        'No available collections. Create one in the Knowledge Base tab first.',
      noKnowledgeBasesConnected: 'No knowledge bases connected to this project yet.',
      mcpToolReadyIntro:
        'This project now exposes an internal MCP tool named ',
      mcpToolReadyOutro:
        '. Any model connected through the project endpoint can call it to search across all connected knowledge bases.',
      otherConnectionOptions: 'Other connection options',
      otherConnectionOptionsDescription:
        'Use these addresses for LAN access, legacy token-in-URL clients, or when the default local URL is not the one you need.',
      duplicateProjectDescription:
        'Create a full copy of this project with a new name, token, servers, integrations, package links, and connected knowledge bases.',
      duplicateProjectNamePlaceholder: 'Client Workspace Copy',
      manageToolsDescription:
        'Enable or disable individual MCP tools for this server. Disabled tools are hidden from the project endpoint and blocked from calls through MCPBox.',
      noServerTools: 'This server does not expose any tools.',
      loadServerToolsError: 'Failed to load server tools',
      updateServerToolsError: 'Failed to update server tools',
      disabledToolsBadge: (count: number) => `${count} tools off`,
      launchProjectDescription:
        'Choose how you want to start working with this project locally.',
      projectPromptDescription:
        'Define the shared instruction that connected MCP clients should receive for this project.',
      ollamaNotInstalled: 'Ollama is not installed or not available in PATH.',
      llamaCppNotInstalled: 'llama-server is not installed or not available in PATH.',
      llamaCppNotConfigured: 'Set MCPBOX_LLAMACPP_MODEL to a local GGUF file to enable llama.cpp launch.',
      llamaCppFilePickerHint: 'The chosen .gguf path will override MCPBOX_LLAMACPP_MODEL for this launch.',
      bearerEndpointProtectionDescription:
        'Require Authorization Bearer token for this project MCP endpoint.',
      bearerTokenGeneratedAfterCreate:
        'Token will be generated by the server after project creation.',
      bearerTokenRegenerated: 'Bearer token regenerated',
    },
  },
  ru: {
    labels: {
      appTitle: 'MCPBox',
      controlCenter: '',
      appDescription: 'Единый сервис для управления MCP-серверами.',
      projects: 'Проекты',
      createProject: 'Создать проект',
      duplicateProject: 'Дублировать проект',
      name: 'Название',
      description: 'Описание',
      prompt: 'Промпт',
      save: 'Сохранить',
      saving: 'Сохранение...',
      projectOverview: 'Обзор проекта',
      servers: 'Серверы',
      running: 'Запущено',
      connectionEndpoint: 'Точка подключения',
      addServer: 'Добавить MCP-сервер',
      serverName: 'Название сервера',
      launchCommand: 'Команда запуска',
      stdio: 'STDIO',
      httpStreaming: 'Потоковая передача HTTP',
      workingDirectory: 'Рабочая директория',
      url: 'URL',
      autoStart: 'Запускать автоматически при старте MCPBox',
      copied: 'Скопировано',
      copyUrl: 'Скопировать URL',
      noProjectSelected: 'Проект не выбран',
      ready: 'Готово',
      notSelected: 'Не выбран',
      notSpecified: 'Не указано',
      stop: 'Остановить',
      start: 'Запустить',
      language: 'Язык',
      english: 'English',
      russian: 'Русский',
      arguments: 'Аргументы',
      environmentVariables: 'Переменные окружения',
      environmentVariablePassthrough: 'Передача переменных окружения',
      bearerTokenEnvironmentVariable: 'Переменная окружения Bearer-токена',
      headers: 'Заголовки',
      headersFromEnvironmentVariables: 'Заголовки из переменных окружения',
      addArgument: 'Добавить аргумент',
      addEnvironmentVariable: 'Добавить переменную окружения',
      addHeader: 'Добавить заголовок',
      addVariable: 'Добавить переменную',
      key: 'Ключ',
      value: 'Значение',
      logs: 'Логи',
      auditLogs: 'Журнал аудита',
      performance: 'Производительность',
      timeWindow: 'Период',
      refresh: 'Обновить',
      launchProject: 'Запустить',
      launchOllama: 'Запустить Ollama',
      launchLlamaCpp: 'Запустить llama.cpp',
      launchLMStudio: 'Добавить в LM Studio',
      ollamaModel: 'Модель Ollama',
      llamaCppModel: 'Модель llama.cpp',
      llamaCppModelPath: 'Путь к модели',
      llamaCppModelName: 'Имя модели',
      pauseProject: 'Приостановить проект',
      resumeProject: 'Возобновить проект',
      paused: 'Приостановлен',
      disabled: 'Отключен',
      disableServer: 'Отключить сервер',
      enableServer: 'Включить сервер',
      remote: 'Удаленный',
      unknownActor: 'неизвестный источник',
      requestControl: 'Контроль запросов',
      filterByProject: 'Фильтр по проекту',
      last5Minutes: 'Последние 5 минут',
      lastHour: 'Последний час',
      last24Hours: 'Последние 24 часа',
      allProjects: 'Все проекты',
      currentProjectOnly: 'Только текущий проект',
      activityOverview: 'Сводка активности',
      hottestProject: 'Самый активный проект',
      hottestServer: 'Самый активный сервер',
      requests: 'запросов',
      errors: 'Ошибки',
      avgLatency: 'Средняя задержка',
      p95Latency: 'P95 задержки',
      trafficIn: 'Входящий трафик',
      trafficOut: 'Исходящий трафик',
      errorRate: 'Доля ошибок',
      noActivity: 'Активности пока нет',
      consoleFeed: 'Поток событий',
      requestVolume: 'Объем запросов',
      latencyTrend: 'Динамика задержки',
      topSlowServers: 'Самые медленные серверы',
      topErrorServers: 'Серверы с ошибками',
      topTrafficServers: 'Самые загруженные серверы',
      recentFailures: 'Последние сбои',
      info: 'Инфо',
      serverInfo: 'Информация о сервере',
      capabilities: 'Возможности',
      tools: 'Инструменты',
      resources: 'Ресурсы',
      prompts: 'Промпты',
      readme: 'README',
      instructions: 'Инструкции',
      protocolVersion: 'Версия протокола',
      version: 'Версия',
      noReadme: 'README рядом с локальным сервером не найден.',
      noTools: 'Инструменты не найдены.',
      noResources: 'Ресурсы не найдены.',
      noPrompts: 'Промпты не найдены.',
      inspectServer: 'Проверить сервер',
      health: 'Состояние',
      healthy: 'Исправен',
      failed: 'Ошибка',
      unknown: 'Неизвестно',
      lastCheck: 'Последняя проверка',
      check: 'Проверить',
      oauth: 'OAuth',
      connected: 'Подключен',
      notConnected: 'Не подключен',
      oauthConnected: 'OAuth подключен',
      market: 'Маркет',
      integrations: 'Интеграции',
      catalog: 'Каталог',
      catalogItems: 'Элементы',
      installed: 'Установлено',
      lastSync: 'Последняя синхронизация',
      externalManifestUrl: 'URL внешнего манифеста',
      manifestSource: 'Источник каталога',
      serverSource: 'Сервер',
      localFileSource: 'Локальный файл',
      chooseFile: 'Выбрать файл',
      syncCatalog: 'Синхронизировать каталог',
      allCategories: 'Все категории',
      command: 'Команда',
      endpoint: 'Endpoint',
      runtime: 'Runtime',
      source: 'Source',
      installModel: 'Модель установки',
      systemDependencies: 'Системные зависимости',
      docs: 'Документация',
      website: 'Сайт',
      generalCategory: 'общее',
      mcpDiscovery: 'mcp discovery',
      installPackage: 'Установить пакет',
      uninstallPackage: 'Удалить пакет',
      packageInstalled: 'Пакет установлен',
      packageNotInstalled: 'Пакет не установлен',
      addToProject: 'Добавить в проект',
      installFirst: 'Сначала установить пакет',
      addedToProject: 'Добавлено в проект',
      notInProject: 'Не в проекте',
      knowledgeBase: 'База знаний',
      collections: 'Коллекции',
      create: 'Создать',
      actions: 'Действия',
      edit: 'Редактировать',
      createCollection: 'Создать коллекцию',
      connect: 'Подключить',
      delete: 'Удалить',
      disconnect: 'Отключить',
      index: 'Индексировать',
      search: 'Поиск',
      connectedKnowledgeBases: 'Подключенные базы знаний',
      toolContract: 'Контракт инструмента',
      mcpToolReady: 'MCP-инструмент готов',
      showMore: 'Показать еще',
      hide: 'Скрыть',
      manageTools: 'Инструменты',
      enabled: 'Включен',
      bearerEndpointProtection: 'Bearer-защита endpoint',
      bearerToken: 'Bearer токен',
      generateToken: 'Сгенерировать токен',
      showToken: 'Показать токен',
      hideToken: 'Скрыть токен',
      copyToken: 'Скопировать токен',
    },
    messages: {
      loadingProjects: 'Загружаю проекты...',
      noProjects: 'Проектов пока нет. Создай первый workspace ниже.',
      projectHelper: 'Проект это логическая группа MCP-серверов для одного окружения.',
      projectNamePlaceholder: 'Клиентский workspace',
      projectDescriptionPlaceholder: 'Для чего нужен этот workspace',
      emptySelection:
        'Создай первый workspace слева, и он станет центром управления своими MCP-серверами.',
      emptySelectionBody:
        'Создай первый workspace слева, и он станет центром управления своими MCP-серверами.',
      overviewFallbackDescription: 'Логическая группа для MCP-клиентов и инструментов.',
      connectionDescription:
        'Используйте endpoint проекта с Authorization: Bearer <project-token>. Все клиенты через этот endpoint получают доступ ко всем включённым MCP-серверам проекта.',
      connectionWarning: (_token: string) => 'Чтобы использовать этот endpoint, добавьте и включите хотя бы один MCP-сервер или подключите базу знаний.',
      addServerDescription:
        'Добавьте один или несколько MCP-серверов, чтобы опубликовать их через общий endpoint проекта.',
      serverNamePlaceholder: 'Filesystem Server',
      launchCommandPlaceholder: 'npx -y @modelcontextprotocol/server-filesystem /path',
      commandPlaceholder: 'uvx mcp-server или node dist/index.js',
      argumentPlaceholder: '--port или ./path',
      workingDirectoryPlaceholder: '/absolute/path или оставить пустым',
      urlPlaceholder: 'https://mcp.example.com/mcp',
      bearerTokenPlaceholder: 'MCP_BEARER_TOKEN',
      envPassthroughPlaceholder: 'OPENAI_API_KEY',
      serverControlDescription:
        'Управляй процессами и доступностью всех MCP-серверов, опубликованных через endpoint проекта.',
      noServers: 'В этом проекте пока нет серверов.',
      workspaceGroupFallback: 'Логическая группа для MCP-клиентов',
      serverCount: (count: number) => `${count} серверов`,
      runningCount: (count: number) => `${count} запущено`,
      createProjectError: 'Не удалось создать проект',
      duplicateProjectError: 'Не удалось продублировать проект',
      loadProjectsError: 'Не удалось загрузить проекты',
      addServerError: 'Не удалось добавить сервер',
      startServerError: 'Не удалось запустить сервер',
      stopServerError: 'Не удалось остановить сервер',
      setProjectPausedError: 'Не удалось обновить состояние проекта',
      setServerEnabledError: 'Не удалось обновить состояние сервера',
      logsDescription: 'Мониторинг запросов и действий управления по проектам и MCP-серверам.',
      performanceDescription:
        'Задержки, ошибки и трафик по MCP-серверам прямо на текущем экране логов.',
      loadingLogs: 'Загружаю логи...',
      loadingMetrics: 'Загружаю метрики...',
      noLogs: 'Записей аудита пока нет.',
      noMetrics:
        'Данных по производительности пока нет. Метрики появятся, когда через MCPBox пойдут запросы.',
      projectTag: (id: number) => `проект #${id}`,
      serverTag: (id: number) => `сервер #${id}`,
      consoleDescription:
        'Компактный поток событий по запросам, подключениям и управляющим действиям.',
      popularityDescription:
        'Кто сейчас получает больше всего MCP-трафика в рамках текущего фильтра.',
      inspectDescription:
        'Живой MCP inspection для локального STDIO сервера и nearby README, если он найден.',
      inspectServerError: 'Не удалось проинспектировать сервер',
      checkServerError: 'Не удалось проверить сервер',
      checkServerHealthy: (name: string) => `${name} успешно прошел проверку.`,
      checkServerFailed: (name: string, reason: string) => `${name} не прошел проверку: ${reason}`,
      requestFailed: (status: number) => `Запрос завершился со статусом ${status}`,
      marketDescription:
        'Синхронизируйте внешний манифест интеграций в SQLite и устанавливайте выбранные элементы в текущий проект как связанные MCP-серверы.',
      searchCatalogPlaceholder: 'Поиск по интеграциям, тегам, runtime и package',
      catalogResultsSummary: (visible: number, total: number) => `Показано ${visible} из ${total} интеграций`,
      notSynced: 'Не синхронизировано',
      advancedModeEnabled:
        'Расширенный режим включён. Нажмите Cmd/Ctrl + Shift + U, чтобы скрыть поле.',
      localCatalogFileSelected: (name: string) => `Выбран файл: ${name}`,
      localCatalogFileMissing: 'Сначала выберите локальный JSON-файл каталога.',
      selectProjectBeforeInstall:
        'Выберите проект в боковой панели перед установкой интеграций.',
      noDescriptionProvided: 'Описание отсутствует.',
      upstreamAuthNotice:
        'Аутентификация выполняется на стороне исходного MCP-сервера. После установки MCP-клиент должен завершить вход при подключении через MCPBox.',
      noValue: 'n/a',
      workingDirectoryValue: (path: string) => `Рабочая директория: ${path}`,
      autoStartAfterInstall: 'Запускается автоматически после установки',
      syncManifestToPopulateCatalog:
        'Синхронизируйте внешний манифест, чтобы заполнить каталог.',
      noIntegrationsInCategory: 'В этой категории пока нет интеграций.',
      installIntegrationTitle: (name: string) => `Установить ${name}`,
      installIntegrationFallbackTitle: 'Установка интеграции',
      installIntegrationDescription:
        'Заполните обязательные параметры подключения перед добавлением этой интеграции в выбранный проект.',
      oneValuePerLine: 'По одному значению на строку',
      installIntegrationAction: 'Установить',
      loadPackagesError: 'Не удалось загрузить пакеты',
      installPackageError: 'Не удалось установить пакет',
      addPackageToProjectError: 'Не удалось добавить пакет в проект',
      uninstallPackageError: 'Не удалось удалить пакет',
      catalogInstallAdded: (name: string) => `${name} добавлен в проект.`,
      catalogHealthCheckPassed: (name: string) => `${name} добавлен и успешно прошел health-check.`,
      catalogHealthCheckFailed: (name: string) => `${name} добавлен, но health-check завершился ошибкой.`,
      catalogHealthCheckFailedWithReason: (name: string, reason: string) => `${name} добавлен, но health-check завершился ошибкой: ${reason}`,
      addPackageDialogTitle: (name: string) => `Добавить ${name} в проект`,
      addPackageDialogFallbackTitle: 'Добавить пакет в проект',
      addPackageDialogDescription:
        'Настройте выбранный экземпляр пакета перед добавлением в текущий проект.',
      sharedInstallMode: 'Общая установка пакета',
      projectInstallMode: 'Установка в рамках проекта',
      multiProjectSupported: 'Можно переиспользовать в нескольких проектах',
      singleProjectOnly: 'Лучше подходит для одного проекта',
      packageUsageCount: (count: number) => `Используется в ${count} проектах`,
      packageInUseCannotUninstall: 'Этот пакет всё ещё подключён хотя бы к одному проекту.',
      systemDependencyVersion: (name: string, version: string) => `${name} ${version}+`,
      systemDependencyRequired: (name: string) => `Нужен ${name}`,
      envSchemaDescription: 'Эти значения будут переданы процессу сервера как переменные окружения.',
      launchOllamaError: 'Не удалось запустить терминал Ollama.',
      launchLlamaCppError: 'Не удалось запустить терминал llama.cpp.',
      launchLMStudioError: 'Не удалось открыть LM Studio.',
      noOllamaModels: 'Локальные модели Ollama не найдены.',
      knowledgeBaseHeroTitle: 'Глобальные коллекции знаний',
      knowledgeBaseHeroDescription:
        'Создавайте переиспользуемые коллекции один раз, индексируйте локальные папки и подключайте их к одному или нескольким проектам. Поддерживаются код, текст, CSV, XLSX, DOCX, PPTX и PDF с текстовым слоем.',
      createKnowledgeBaseTitle: 'Создать базу знаний',
      createKnowledgeBaseDescription:
        'Добавьте глобальную коллекцию, сразу выберите папку с файлами, и MCPBox сразу её проиндексирует. Можно индексировать код, текст, таблицы, офисные документы и PDF с текстовым слоем.',
      editKnowledgeBaseDescription:
        'Обновите название, папку с файлами или настройку автопереиндексации. При сохранении индекс будет пересобран из выбранной папки.',
      collectionIdLabel: 'ID коллекции',
      collectionIdPlaceholder: 'crm_gym',
      collectionNamePlaceholder: 'CRM Gym Codebase',
      indexPathLabel: 'Путь индекса',
      indexPathPlaceholder: '.mcpbox/rag/crm_gym.bleve',
      sourceFolderTitle: 'Папка с файлами',
      sourceFolderPlaceholder: '/Users/artur/projects/crm-gym',
      autoReindexTitle: 'Переиндексировать каждые 10 минут',
      autoReindexDescription:
        'Если включено, MCPBox будет автоматически пересобирать эту коллекцию каждые 10 минут, пока приложение запущено.',
      autoReindexBadge: 'Автопереиндексация: каждые 10 мин',
      noKnowledgeBasesCreated: 'Базы знаний еще не созданы.',
      indexFolderTitle: 'Папка с файлами',
      indexFolderDescription:
        'Укажите локальную папку, файлы из которой нужно добавить в эту базу знаний. Системные папки вроде node_modules, vendor, build-артефактов и Python virtual environment будут пропущены автоматически.',
      indexFolderPlaceholder: '/path/to/project',
      supportedFormatsLabel: 'Поддерживается',
      supportedFormatsValue: 'Код, Текст, CSV, XLSX, DOCX, PPTX, PDF (с текстовым слоем)',
      searchCollectionTitle: 'Поиск по коллекции',
      searchCollectionDescription:
        'Выполните быстрый поиск по ключевым словам и просмотрите наиболее релевантные проиндексированные фрагменты.',
      searchCollectionPlaceholder: 'payment gateway',
      searchResultsTitle: 'Результаты поиска',
      searchResultsDescription: (name: string) => `Лучшие совпадения в коллекции ${name}.`,
      searchResultsEmpty: 'По этому запросу совпадений не найдено.',
      searchQueryRequired: 'Нужно указать поисковый запрос.',
      deleteKnowledgeBaseConfirm: 'Удалить эту базу знаний?',
      connectedKnowledgeBasesDescription:
        'Подключите к этому проекту одну или несколько глобальных коллекций.',
      connectKnowledgeBaseTitle: 'Подключить базу знаний',
      connectKnowledgeBaseDescription:
        'Выберите одну из глобальных коллекций и подключите ее к этому проекту.',
      noAvailableCollections:
        'Нет доступных коллекций. Сначала создайте одну во вкладке "База знаний".',
      noKnowledgeBasesConnected: 'К этому проекту еще не подключены базы знаний.',
      mcpToolReadyIntro:
        'Этот проект теперь публикует внутренний MCP-инструмент ',
      mcpToolReadyOutro:
        '. Любая модель, подключенная через endpoint проекта, может вызывать его для поиска по всем подключенным базам знаний.',
      otherConnectionOptions: 'Другие варианты подключения',
      otherConnectionOptionsDescription:
        'Используйте эти адреса для подключения по локальной сети, для legacy-клиентов с токеном в URL или если нужен не основной локальный URL.',
      duplicateProjectDescription:
        'Создать полную копию этого проекта с новым именем, token, серверами, интеграциями, package links и подключёнными базами знаний.',
      duplicateProjectNamePlaceholder: 'Копия Client Workspace',
      manageToolsDescription:
        'Включайте и выключайте отдельные MCP tools у этого сервера. Выключенные tools скрываются из project endpoint и блокируются для вызова через MCPBox.',
      noServerTools: 'Этот сервер не публикует tools.',
      loadServerToolsError: 'Не удалось загрузить tools сервера',
      updateServerToolsError: 'Не удалось обновить tools сервера',
      disabledToolsBadge: (count: number) => `${count} tools выкл`,
      launchProjectDescription:
        'Выберите, как вы хотите запускать этот проект локально.',
      projectPromptDescription:
        'Задайте общую инструкцию, которую подключенные MCP-клиенты должны получать для этого проекта.',
      ollamaNotInstalled: 'Ollama не установлена или недоступна в PATH.',
      llamaCppNotInstalled: 'llama-server не установлен или недоступен в PATH.',
      llamaCppNotConfigured: 'Укажите MCPBOX_LLAMACPP_MODEL на локальный GGUF-файл, чтобы включить запуск llama.cpp.',
      llamaCppFilePickerHint: 'Выбранный путь к .gguf переопределит MCPBOX_LLAMACPP_MODEL только для этого запуска.',
      bearerEndpointProtectionDescription:
        'Требовать Bearer токен для MCP endpoint этого проекта.',
      bearerTokenGeneratedAfterCreate:
        'Токен будет сгенерирован сервером после создания проекта.',
      bearerTokenRegenerated: 'Bearer токен пересоздан',
    },
  },
};

export function detectInitialLanguage(): Language {
  if (typeof window === 'undefined') {
    return defaultLanguage;
  }

  const storedLanguage = window.localStorage.getItem(languageStorageKey);
  if (storedLanguage === 'en' || storedLanguage === 'ru') {
    return storedLanguage;
  }

  return defaultLanguage;
}

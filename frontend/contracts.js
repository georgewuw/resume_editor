/**
 * Shared frontend module contracts.
 *
 * @typedef {{fields: Record<string, string[]>}} ResumeContent
 * @typedef {{index: number, key: string, group: string, groupTitle: string,
 *   groupEn: string, label: string, type: string, multiline: boolean, hint?: string, placeholder?: string}} ResumeField
 * @typedef {{id: string, title: string, en: string, description: string,
 *   items?: ResumeField[]}} ResumeGroup
 * @typedef {{code: string, message: string}} ApplicationError
 * @typedef {{level: string, event: string, code?: string}} ClientLog
 * @typedef {{filename: string, company: string, role: string, jd: string, modifiedAt: string}} DocumentInfo
 * @typedef {{document: DocumentInfo, content: ResumeContent}} ResumeDocument
 * @typedef {{company: string, role: string, jd: string, source: string}} NewDocument
 * @typedef {{filled: number, total: number, groups: Record<string, number>}} ContentProgress
 * @typedef {{platform: string,
 *   workspace: () => Promise<{documents: DocumentInfo[], current: ResumeDocument}>,
 *   listDocuments: () => Promise<DocumentInfo[]>,
 *   openDocument: (filename: string) => Promise<ResumeDocument>,
 *   createDocument: (input: NewDocument) => Promise<ResumeDocument>,
 *   deleteDocument: (filename: string) => Promise<{documents: DocumentInfo[], current: ResumeDocument}>,
 *   saveDocument: (filename: string, content: ResumeContent, jd: string) => Promise<DocumentInfo>,
 *   progress: (content: ResumeContent) => Promise<ContentProgress>,
 *   load: () => Promise<ResumeContent>, fill: (content: ResumeContent) => Promise<string>,
 *   save: (content: ResumeContent) => Promise<string>, export: (filename: string, content: ResumeContent) => Promise<string>,
 *   log: (entry: ClientLog) => Promise<void>}} ResumeBackend
 * @typedef {{metadata: {name: string, version: string}, schema: ResumeField[],
 *   groups: ResumeGroup[], initial: ResumeContent, errors: Record<string, string>}} ApplicationData
 */

export {};

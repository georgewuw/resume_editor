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
 * @typedef {{platform: string,
 *   load: () => Promise<ResumeContent>, fill: (content: ResumeContent) => Promise<string>,
 *   save: (content: ResumeContent) => Promise<string>, export: (content: ResumeContent) => Promise<string>,
 *   log: (entry: ClientLog) => Promise<void>}} ResumeBackend
 * @typedef {{metadata: {name: string, version: string}, schema: ResumeField[],
 *   groups: ResumeGroup[], initial: ResumeContent, errors: Record<string, string>}} ApplicationData
 */

export {};

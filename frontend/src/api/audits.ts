import { requestPage } from './client'
import type { AuditEvent } from '../types/audit'

export const listAudits = () => requestPage<AuditEvent>('/audits?page=1&page_size=100')

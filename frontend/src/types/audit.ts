export interface AuditEvent {
  id: number
  request_id: string
  user_id: number
  actor_email: string
  action: string
  entity_type: string
  entity_id: number
  before_json: string
  after_json: string
  created_at: string
}

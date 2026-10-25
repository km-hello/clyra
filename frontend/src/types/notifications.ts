export interface NotificationRun {
  id: string
  run_date: string
  scheduled_time: string | null
  run_type: 'scheduled' | 'manual_test'
  channel: 'telegram'
  status: 'running' | 'succeeded' | 'failed'
  matched_count: number
  message_count: number
  started_at: number
  finished_at: number | null
  error_message: string | null
}

export interface NotificationRunsResponse {
  data: NotificationRun[]
  meta: {
    timezone: string
  }
}

export interface NotificationTestResponse {
  data: { run_id: string; message_count: number; status: 'succeeded' }
}

// Generated types from OpenAPI spec

export interface ErrorResponse {
  error_code: string;
  error_message: string;
}

export interface PaginationMeta {
  limit: number;
  offset: number;
  total: number;
}

export interface PaginatedResponse<T> {
  items: T[];
  pagination: PaginationMeta;
}

// User
export interface UserDTO {
  id: string;
  email: string;
  name: string;
  locale: string;
  created_at: string;
}

// Auth
export interface AuthTokens {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  locale?: string;
}

export interface LoginResponse {
  user: UserDTO;
  tokens: AuthTokens;
}

export interface RegisterResponse {
  message: string;
}

export interface RefreshRequest {
  refresh_token: string;
}

export interface VerifyEmailRequest {
  email: string;
  code: string;
}

export interface ResendVerificationRequest {
  email: string;
}

export interface ForgotPasswordRequest {
  email: string;
}

export interface ResetPasswordRequest {
  email: string;
  code: string;
  password: string;
}

// Application Status
export type ApplicationStatus =
  | "active"
  | "on_hold"
  | "rejected"
  | "offer"
  | "archived";

// Nested DTOs for Application
export interface JobNestedDTO {
  id: string;
  title: string;
  company?: CompanyDTO;
}

export interface ResumeNestedDTO {
  id: string;
  name: string;
  type: "uploaded" | "builder";
}

// Application
export interface ApplicationDTO {
  id: string;
  name: string;
  status: ApplicationStatus;
  applied_at: string;
  created_at: string;
  updated_at: string;
  last_activity_at?: string;
  current_stage_id?: string;
  current_stage_name?: string;
  job?: JobNestedDTO;
  resume?: ResumeNestedDTO;
  application_comments?: CommentDTO[];
  stage_comments?: CommentDTO[];
}

export interface CreateApplicationRequest {
  job_id: string;
  resume_id?: string;
  resume_builder_id?: string;
  name: string;
  applied_at?: string;
}

export interface UpdateApplicationRequest {
  status?: string;
}

// Application Stage
export interface ApplicationStageDTO {
  id: string;
  application_id: string;
  stage_template_id: string;
  stage_name: string;
  order: number;
  status: string;
  started_at: string;
  completed_at?: string;
  created_at: string;
}

export interface AddStageRequest {
  stage_template_id: string;
  comment?: string;
}

export interface CompleteStageRequest {
  completed_at?: string;
}

export interface UpdateStageRequest {
  status?: "pending" | "active" | "completed" | "skipped" | "cancelled";
  completed_at?: string;
}

// Stage Template
export interface StageTemplateDTO {
  id: string;
  name: string;
  order: number;
  created_at: string;
}

export interface CreateStageTemplateRequest {
  name: string;
  order: number;
}

export interface UpdateStageTemplateRequest {
  name?: string;
  order?: number;
}

// Company
export interface CompanyDTO {
  id: string;
  name: string;
  location?: string;
  notes?: string;
  is_favorite: boolean;
  created_at: string;
  updated_at: string;
  applications_count: number;
  active_applications_count: number;
  derived_status: "idle" | "active" | "interviewing";
  last_activity_at?: string;
}

export interface CreateCompanyRequest {
  name: string;
  location?: string;
  notes?: string;
}

export interface UpdateCompanyRequest {
  name?: string;
  location?: string;
  notes?: string;
}

// Job
export interface JobDTO {
  id: string;
  title: string;
  company_id?: string;
  company_name?: string;
  url?: string;
  source?: string;
  notes?: string;
  description?: string;
  status: "active" | "archived";
  is_favorite: boolean;
  applications_count: number;
  created_at: string;
  updated_at: string;
}

export interface CreateJobRequest {
  title: string;
  company_id?: string;
  url?: string;
  source?: string;
  notes?: string;
  description?: string;
}

export interface UpdateJobRequest {
  title?: string;
  company_id?: string;
  url?: string;
  source?: string;
  notes?: string;
  description?: string;
  status?: "active" | "archived";
}

// Match Score
export interface MatchScoreCategory {
  name: string;
  score: number;
  details: string;
}

export interface MatchScoreResponse {
  overall_score: number;
  categories: MatchScoreCategory[];
  missing_keywords: string[];
  strengths: string[];
  summary: string;
  from_cache: boolean;
}

// Calendar
export interface CalendarStatusDTO {
  connected: boolean;
  email?: string;
}

export interface OAuthURLResponse {
  url: string;
}

export interface CreateCalendarEventRequest {
  stage_id: string;
  title: string;
  start_time: string;
  duration_min: number;
  description?: string;
}

export interface CalendarEventDTO {
  event_id: string;
  stage_id: string;
  title: string;
  start_time: string;
  end_time: string;
  link?: string;
}

// Resume
export type StorageType = "external" | "s3";

export interface ResumeDTO {
  id: string;
  title: string;
  file_url: string | null;
  storage_type: StorageType;
  storage_key?: string | null;
  is_active: boolean;
  applications_count: number;
  can_delete: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateResumeRequest {
  title: string;
  file_url?: string | null;
  is_active?: boolean;
}

export interface UpdateResumeRequest {
  title?: string;
  file_url?: string | null;
  is_active?: boolean;
}

export interface GenerateUploadURLRequest {
  filename: string;
  content_type: string;
}

export interface GenerateUploadURLResponse {
  resume_id: string;
  upload_url: string;
  expires_in: number;
}

export interface DownloadURLResponse {
  download_url: string;
  expires_in: number;
}

// Comment
export interface CommentDTO {
  id: string;
  application_id: string;
  stage_id?: string;
  content: string;
  created_at: string;
}

export interface CreateCommentRequest {
  application_id: string;
  stage_id?: string;
  content: string;
}

// Subscription
export interface PlanLimits {
  max_jobs: number; // -1 = unlimited
  max_resumes: number;
  max_applications: number;
  max_ai_requests: number;
  max_job_parses: number;
  max_resume_builders: number;
  max_cover_letters: number;
}

export interface SubscriptionUsage {
  jobs: number;
  resumes: number;
  applications: number;
  ai_requests: number;
  job_parses: number;
  resume_builders: number;
  cover_letters: number;
}

export type SubscriptionPlan = "free" | "pro" | "enterprise";
export type SubscriptionStatus =
  | "free"
  | "active"
  | "past_due"
  | "cancelled"
  | "paused";

export interface SubscriptionDTO {
  plan: SubscriptionPlan;
  status: SubscriptionStatus;
  limits: PlanLimits;
  usage: SubscriptionUsage;
  current_period_end?: string;
  cancel_at?: string;
}

export interface CheckoutConfigDTO {
  client_token: string;
  prices: Record<string, string>;
  environment: string;
}

export interface PortalSessionDTO {
  url: string;
}

// Health
export interface HealthResponse {
  status: string;
  version: string;
  services: Record<string, string>;
}

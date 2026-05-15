export interface ApiErrorPayload {
  error_code: string;
  message: string;
}

export interface ApiResponse<T> {
  success: boolean;
  message: string;
  data: T;
  error?: ApiErrorPayload;
  request_id?: string;
}

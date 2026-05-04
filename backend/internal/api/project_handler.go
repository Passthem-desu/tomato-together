package api

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"tomatogether/backend/internal/models"
	"tomatogether/backend/internal/service"
)

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	claims := s.validateL3Token(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "week"
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"period":          period,
		"total_pomodoros": 0,
		"total_duration":  0,
		"by_project":      []interface{}{},
		"by_day":          []interface{}{},
	}))
}

func (s *Server) handleGetProjects(w http.ResponseWriter, r *http.Request) {
	claims := s.validateL3Token(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	projectService := s.projectService.(*service.ProjectService)
	projects, err := projectService.GetProjectsByUser(claims.UserID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "get_projects_failed")
		return
	}

	result := make([]map[string]interface{}, 0, len(projects))
	for _, p := range projects {
		result = append(result, map[string]interface{}{
			"id":         p.ID,
			"name":       p.Name,
			"created_at": p.CreatedAt.Format(time.RFC3339),
		})
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"projects": result,
	}))
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	claims := s.validateL3Token(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &req); err != nil || req.Name == "" {
		s.errorResponse(w, http.StatusBadRequest, "name_required")
		return
	}

	projectService := s.projectService.(*service.ProjectService)
	project, err := projectService.CreateProject(claims.UserID, req.Name)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "create_project_failed")
		return
	}

	s.jsonResponse(w, http.StatusCreated, formatSuccess(map[string]interface{}{
		"id":         project.ID,
		"name":       project.Name,
		"created_at": project.CreatedAt.Format(time.RFC3339),
	}))
}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	claims := s.validateL3Token(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	vars := mux.Vars(r)
	projectID := vars["id"]

	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &req); err != nil || req.Name == "" {
		s.errorResponse(w, http.StatusBadRequest, "name_required")
		return
	}

	projectService := s.projectService.(*service.ProjectService)
	if err := projectService.UpdateProject(projectID, req.Name); err != nil {
		if err == service.ErrProjectNotFound {
			s.errorResponse(w, http.StatusNotFound, "project_not_found")
		} else {
			s.errorResponse(w, http.StatusInternalServerError, "update_project_failed")
		}
		return
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"id":         projectID,
		"name":       req.Name,
		"updated_at": time.Now().Format(time.RFC3339),
	}))
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	claims := s.validateL3Token(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	vars := mux.Vars(r)
	projectID := vars["id"]

	projectService := s.projectService.(*service.ProjectService)
	if err := projectService.DeleteProject(projectID); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "delete_project_failed")
		return
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]string{
		"message": "项目已删除",
	}))
}

func (s *Server) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	claims := s.validateL3Token(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	status := r.URL.Query().Get("status")
	projectID := r.URL.Query().Get("project_id")

	taskService := s.taskService.(*service.TaskService)
	var tasks []*models.Task
	var err error

	if projectID != "" {
		tasks, err = taskService.GetTasksByProject(claims.UserID, projectID)
	} else {
		tasks, err = taskService.GetTasksByUser(claims.UserID, status)
	}

	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "get_tasks_failed")
		return
	}

	result := make([]map[string]interface{}, 0, len(tasks))
	for _, t := range tasks {
		item := map[string]interface{}{
			"id":         t.ID,
			"client_id":  t.ClientID,
			"title":      t.Title,
			"status":     t.Status,
			"created_at": t.CreatedAt.Format(time.RFC3339),
		}
		if t.ProjectID != nil {
			item["project_id"] = *t.ProjectID
		}
		if t.CompletedAt != nil {
			item["completed_at"] = t.CompletedAt.Format(time.RFC3339)
		}
		result = append(result, item)
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"tasks": result,
	}))
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	claims := s.validateL3Token(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	var req struct {
		ClientID  string  `json:"client_id"`
		Title     string  `json:"title"`
		ProjectID *string `json:"project_id"`
	}
	if err := decodeJSON(r, &req); err != nil || req.Title == "" {
		s.errorResponse(w, http.StatusBadRequest, "title_required")
		return
	}

	taskService := s.taskService.(*service.TaskService)
	task, err := taskService.CreateTask(claims.UserID, req.ClientID, req.Title, req.ProjectID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "create_task_failed")
		return
	}

	s.jsonResponse(w, http.StatusCreated, formatSuccess(map[string]interface{}{
		"id":         task.ID,
		"client_id":  task.ClientID,
		"title":      task.Title,
		"status":     task.Status,
		"project_id": task.ProjectID,
		"created_at": task.CreatedAt.Format(time.RFC3339),
	}))
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	claims := s.validateL3Token(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	vars := mux.Vars(r)
	taskID := vars["id"]

	var req struct {
		Title     *string `json:"title"`
		Status    *string `json:"status"`
		ProjectID *string `json:"project_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid_request")
		return
	}

	taskService := s.taskService.(*service.TaskService)
	if err := taskService.UpdateTask(taskID, req.Title, req.Status, req.ProjectID); err != nil {
		if err == service.ErrTaskNotFound {
			s.errorResponse(w, http.StatusNotFound, "task_not_found")
		} else {
			s.errorResponse(w, http.StatusInternalServerError, "update_task_failed")
		}
		return
	}

	task, _ := taskService.GetTask(taskID)
	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"id":           task.ID,
		"client_id":    task.ClientID,
		"title":        task.Title,
		"status":       task.Status,
		"project_id":   task.ProjectID,
		"completed_at": task.CompletedAt,
	}))
}

func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	claims := s.validateL3Token(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	vars := mux.Vars(r)
	taskID := vars["id"]

	taskService := s.taskService.(*service.TaskService)
	if err := taskService.DeleteTask(taskID); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "delete_task_failed")
		return
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]string{
		"message": "WIP 已删除",
	}))
}

func (s *Server) handleSyncTasks(w http.ResponseWriter, r *http.Request) {
	claims := s.validateL3Token(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "token_required")
		return
	}

	var req struct {
		Tasks []struct {
			ClientID  string  `json:"client_id"`
			Title     string  `json:"title"`
			Status    string  `json:"status"`
			ProjectID *string `json:"project_id"`
			CreatedAt string  `json:"created_at"`
		} `json:"tasks"`
	}
	if err := decodeJSON(r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid_request")
		return
	}

	tasks := make([]*models.Task, len(req.Tasks))
	for i, t := range req.Tasks {
		tasks[i] = &models.Task{
			ClientID:  t.ClientID,
			Title:     t.Title,
			Status:    t.Status,
			ProjectID: t.ProjectID,
		}
	}

	taskService := s.taskService.(*service.TaskService)
	mapping, err := taskService.SyncTasks(claims.UserID, tasks)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "sync_tasks_failed")
		return
	}

	result := make([]map[string]string, 0, len(mapping))
	for clientID, serverID := range mapping {
		result = append(result, map[string]string{
			"client_id": clientID,
			"server_id": serverID,
		})
	}

	s.jsonResponse(w, http.StatusOK, formatSuccess(map[string]interface{}{
		"synced": len(mapping),
		"tasks":  result,
	}))
}
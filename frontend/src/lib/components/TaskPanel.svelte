<script lang="ts">
  import { tasksStore } from '$lib/stores';
  import { tasks as tasksApi } from '$lib/api/client';
  import type { Task } from '$lib/types';
  import { browser } from '$app/environment';
  import { onMount } from 'svelte';

  interface Props {
    projectId?: string | null;
  }

  let { projectId = null }: Props = $props();

  let newTaskTitle = $state('');
  let loading = $state(false);

  let filteredTasks = $derived(projectId 
    ? $tasksStore.filter(t => t.project_id === projectId)
    : $tasksStore);

  let todoTasks = $derived(filteredTasks.filter(t => t.status === 'TODO'));
  let wipTasks = $derived(filteredTasks.filter(t => t.status === 'WIP'));
  let doneTasks = $derived(filteredTasks.filter(t => t.status === 'DONE'));

  async function addTask() {
    if (!newTaskTitle.trim()) return;
    
    const clientId = crypto.randomUUID();
    const newTask: Task = {
      id: clientId,
      client_id: clientId,
      title: newTaskTitle.trim(),
      status: 'TODO',
      project_id: projectId || undefined,
      created_at: new Date().toISOString()
    };

    tasksStore.update(tasks => [...tasks, newTask]);
    newTaskTitle = '';

    saveToLocalStorage();

    try {
      const token = browser ? localStorage.getItem('tomatogether_token') : null;
      if (token) {
        await tasksApi.create(clientId, newTask.title, projectId || undefined);
      }
    } catch (e) {
      // 离线模式
    }
  }

  async function updateTaskStatus(task: Task, newStatus: 'TODO' | 'WIP' | 'DONE') {
    tasksStore.update(tasks => 
      tasks.map(t => t.id === task.id ? { ...t, status: newStatus } : t)
    );
    
    saveToLocalStorage();

    try {
      const token = browser ? localStorage.getItem('tomatogether_token') : null;
      if (token) {
        await tasksApi.update(task.id, { status: newStatus });
      }
    } catch (e) {
      // 离线模式
    }
  }

  async function deleteTask(task: Task) {
    tasksStore.update(tasks => tasks.filter(t => t.id !== task.id));
    saveToLocalStorage();

    try {
      const token = browser ? localStorage.getItem('tomatogether_token') : null;
      if (token && !task.client_id) {
        await tasksApi.delete(task.id);
      }
    } catch (e) {
      // 离线模式
    }
  }

  function saveToLocalStorage() {
    if (browser) {
      localStorage.setItem('tomatogether_tasks', JSON.stringify($tasksStore));
    }
  }

  function loadFromLocalStorage() {
    if (browser) {
      const stored = localStorage.getItem('tomatogether_tasks');
      if (stored) {
        try {
          const tasks = JSON.parse(stored);
          tasksStore.set(tasks);
        } catch (e) {
          console.error('Failed to load tasks from localStorage:', e);
        }
      }
    }
  }

  function handleKeyPress(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      addTask();
    }
  }

  onMount(() => {
    loadFromLocalStorage();
  });
</script>

<div class="task-panel">
  <h3 class="title">
    <span class="icon">📋</span>
    待办事项
  </h3>

  <div class="add-task">
    <input 
      type="text" 
      bind:value={newTaskTitle}
      placeholder="添加新任务..."
      onkeypress={handleKeyPress}
    />
    <button onclick={addTask}>添加</button>
  </div>

  <div class="task-sections">
    <div class="task-section">
      <h4 class="section-title">
        <span class="dot todo"></span>
        待处理 ({todoTasks.length})
      </h4>
      <ul class="task-list">
        {#each todoTasks as task (task.id)}
          <li class="task-item">
            <span class="task-title">{task.title}</span>
            <div class="task-actions">
              <button onclick={() => updateTaskStatus(task, 'WIP')}>→</button>
              <button class="delete" onclick={() => deleteTask(task)}>×</button>
            </div>
          </li>
        {/each}
      </ul>
    </div>

    <div class="task-section">
      <h4 class="section-title">
        <span class="dot wip"></span>
        进行中 ({wipTasks.length})
      </h4>
      <ul class="task-list">
        {#each wipTasks as task (task.id)}
          <li class="task-item">
            <span class="task-title">{task.title}</span>
            <div class="task-actions">
              <button onclick={() => updateTaskStatus(task, 'DONE')}>✓</button>
              <button onclick={() => updateTaskStatus(task, 'TODO')}>←</button>
            </div>
          </li>
        {/each}
      </ul>
    </div>

    <div class="task-section">
      <h4 class="section-title">
        <span class="dot done"></span>
        已完成 ({doneTasks.length})
      </h4>
      <ul class="task-list done">
        {#each doneTasks as task (task.id)}
          <li class="task-item">
            <span class="task-title">{task.title}</span>
            <div class="task-actions">
              <button onclick={() => updateTaskStatus(task, 'WIP')}>↺</button>
              <button class="delete" onclick={() => deleteTask(task)}>×</button>
            </div>
          </li>
        {/each}
      </ul>
    </div>
  </div>
</div>

<style>
  .task-panel {
    background: var(--color-bg-secondary);
    border-radius: 0.5rem;
    padding: 1rem;
    min-width: 280px;
  }

  .title {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin: 0 0 1rem 0;
    font-size: 1rem;
    color: var(--color-text);
  }

  .icon {
    font-size: 1.25rem;
  }

  .add-task {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .add-task input {
    flex: 1;
    padding: 0.5rem;
    border: 1px solid var(--color-border);
    border-radius: 0.25rem;
    background: var(--color-bg);
    color: var(--color-text);
  }

  .add-task button {
    padding: 0.5rem 1rem;
    background: var(--color-primary);
    color: white;
    border: none;
    border-radius: 0.25rem;
    cursor: pointer;
  }

  .add-task button:hover {
    background: var(--color-primary-dark);
  }

  .task-sections {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .task-section {
    background: var(--color-bg);
    border-radius: 0.25rem;
    padding: 0.75rem;
  }

  .section-title {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin: 0 0 0.5rem 0;
    font-size: 0.875rem;
    color: var(--color-text-secondary);
  }

  .dot {
    width: 0.5rem;
    height: 0.5rem;
    border-radius: 50%;
  }

  .dot.todo { background: #ff9800; }
  .dot.wip { background: #2196f3; }
  .dot.done { background: #4caf50; }

  .task-list {
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .task-list.done {
    opacity: 0.7;
  }

  .task-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.5rem;
    border-radius: 0.25rem;
    margin-bottom: 0.25rem;
    background: var(--color-bg-secondary);
  }

  .task-title {
    flex: 1;
    color: var(--color-text);
    font-size: 0.875rem;
  }

  .task-actions {
    display: flex;
    gap: 0.25rem;
  }

  .task-actions button {
    width: 1.5rem;
    height: 1.5rem;
    border: none;
    border-radius: 0.25rem;
    background: var(--color-primary);
    color: white;
    font-size: 0.75rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .task-actions button.delete {
    background: #999;
  }

  .task-actions button:hover {
    opacity: 0.8;
  }

  :global(:root) {
    --color-primary: #ff6b6b;
    --color-primary-dark: #ee5a5a;
    --color-bg: #1a1a2e;
    --color-bg-secondary: #16213e;
    --color-text: #eaeaea;
    --color-text-secondary: #a0a0a0;
    --color-border: #333;
  }

  @media (prefers-color-scheme: light) {
    :global(:root) {
      --color-bg: #f5f5f5;
      --color-bg-secondary: #ffffff;
      --color-text: #333;
      --color-text-secondary: #666;
      --color-border: #ddd;
    }
  }
</style>
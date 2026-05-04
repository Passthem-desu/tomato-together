<script lang="ts">
  import { roomMembersStore } from '$lib/stores';
  import type { RoomMember } from '$lib/types';

  interface Props {
    onFollow?: (userId: string) => void;
  }

  let { onFollow = () => {} }: Props = $props();

  function getPomodoroStatus(member: RoomMember): string {
    if (!member.pomodoro?.is_active) return '';
    
    if (member.pomodoro.is_following) {
      return `跟随 ${member.pomodoro.leader_username}`;
    }
    
    if (member.pomodoro.remaining_seconds) {
      const mins = Math.floor(member.pomodoro.remaining_seconds / 60);
      return `${mins} 分钟`;
    }
    
    return '专注中';
  }

  function getStatusIcon(member: RoomMember): string {
    if (!member.is_online) return '⚫';
    if (member.pomodoro?.is_active) return '🍅';
    if (member.status?.emoji) return member.status.emoji;
    return '👤';
  }
</script>

<div class="user-list">
  <h3 class="title">
    <span class="icon">👥</span>
    房间成员
    <span class="count">{$roomMembersStore.length}</span>
  </h3>

  <ul class="users">
    {#each $roomMembersStore as member (member.id)}
      <li class="user-item" class:offline={!member.is_online}>
        <div class="user-avatar">
          {getStatusIcon(member)}
        </div>
        
        <div class="user-info">
          <div class="user-name">
            {member.username}
            {#if member.is_owner}
              <span class="owner-badge">👑</span>
            {/if}
          </div>
          
          {#if member.status?.message}
            <div class="user-status">{member.status.message}</div>
          {/if}
          
          {#if member.pomodoro?.is_active}
            <div class="user-pomodoro">
              {getPomodoroStatus(member)}
            </div>
          {/if}
        </div>

        {#if member.pomodoro?.is_active && !member.pomodoro.is_following}
          <button 
            class="btn-follow" 
            onclick={() => onFollow(member.id)}
            title="跟随此用户"
          >
            跟随
          </button>
        {/if}
      </li>
    {/each}
  </ul>

  {#if $roomMembersStore.length === 0}
    <div class="empty-state">
      还没有成员加入
    </div>
  {/if}
</div>

<style>
  .user-list {
    background: var(--color-bg-secondary);
    border-radius: 0.5rem;
    padding: 1rem;
    min-width: 250px;
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

  .count {
    background: var(--color-primary);
    color: white;
    padding: 0.125rem 0.5rem;
    border-radius: 1rem;
    font-size: 0.75rem;
  }

  .users {
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .user-item {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem;
    border-radius: 0.5rem;
    transition: background 0.2s;
  }

  .user-item:hover {
    background: rgba(255, 255, 255, 0.05);
  }

  .user-item.offline {
    opacity: 0.5;
  }

  .user-avatar {
    font-size: 1.5rem;
    width: 2.5rem;
    height: 2.5rem;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-bg);
    border-radius: 50%;
  }

  .user-info {
    flex: 1;
    min-width: 0;
  }

  .user-name {
    font-weight: 500;
    color: var(--color-text);
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .owner-badge {
    font-size: 0.75rem;
  }

  .user-status {
    font-size: 0.75rem;
    color: var(--color-text-secondary);
    margin-top: 0.125rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .user-pomodoro {
    font-size: 0.75rem;
    color: var(--color-primary);
    margin-top: 0.125rem;
  }

  .btn-follow {
    padding: 0.25rem 0.75rem;
    font-size: 0.75rem;
    background: var(--color-primary);
    color: white;
    border: none;
    border-radius: 1rem;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-follow:hover {
    background: var(--color-primary-dark);
  }

  .empty-state {
    text-align: center;
    color: var(--color-text-secondary);
    padding: 2rem;
  }

  :global(:root) {
    --color-primary: #ff6b6b;
    --color-primary-dark: #ee5a5a;
    --color-bg: #1a1a2e;
    --color-bg-secondary: #16213e;
    --color-text: #eaeaea;
    --color-text-secondary: #a0a0a0;
  }

  @media (prefers-color-scheme: light) {
    :global(:root) {
      --color-bg: #f5f5f5;
      --color-bg-secondary: #e0e0e0;
      --color-text: #333;
      --color-text-secondary: #666;
    }
  }
</style>
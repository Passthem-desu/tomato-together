<script lang="ts">
  import AuthForm from '$lib/components/AuthForm.svelte';
  import { goto } from '$app/navigation';
  import { browser } from '$app/environment';
  import { onMount } from 'svelte';
  import { userStore } from '$lib/stores';

  onMount(() => {
    if (browser && $userStore) {
      goto('/');
    }
  });
</script>

<div class="register-page">
  <AuthForm />
</div>

<style>
  .register-page {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 60vh;
  }
</style>
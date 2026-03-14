<script>
  import { onMount } from "svelte";
  import {
    GetInfo,
    Upload,
    Cancel,
  } from "../bindings/github.com/nobonobo/obs-uploader/appservice.js";

  /** @type {import("../bindings/github.com/nobonobo/obs-uploader/models.js").Fields} */
  const id = new URLSearchParams(location.search).get("id");
  let info = {};
  let fields = [];
  let formData = {};
  let submitting = false;

  onMount(async () => {
    try {
      info = await GetInfo(id);
      console.log(info);
      fields = info.fields;
      // Initialize formData with default values
      for (const field of fields) {
        formData[field.id] = field.default;
      }
    } catch (err) {
      console.error("Failed to load field definitions:", err);
    }
  });

  const handleUpload = async () => {
    if (submitting) return;
    submitting = true;
    try {
      await Upload(id, formData);
    } catch (err) {
      console.error("Upload failed:", err);
      submitting = false;
    }
  };

  const handleCancel = async () => {
    try {
      await Cancel(id);
    } catch (err) {
      console.error("Cancel failed:", err);
    }
  };
</script>

<div
  class="card w-full min-h-screen flex flex-col justify-center items-center preset-filled-surface-100-900 p-4 text-center"
>
  <div class="w-full max-w-md">
    <h1>Uploading Details</h1>

    {#if fields.length === 0}
      <p>Loading form...</p>
    {:else}
      <form class="dynamic-form" on:submit|preventDefault={handleUpload}>
        <div class="form-group">
          <label for="outputPath">Output Path</label>
          <input
            id="outputPath"
            type="text"
            readonly
            disabled={submitting}
            value={info.outputPath}
            class="opacity-70 cursor-not-allowed"
          />
        </div>

        {#each fields as field (field.id)}
          <div class="form-group">
            <label for={field.id}>{field.name}</label>

            {#if field.type === "textarea"}
              <textarea
                id={field.id}
                bind:value={formData[field.id]}
                rows="4"
                disabled={submitting}
              ></textarea>
            {:else if field.type === "number"}
              <input
                id={field.id}
                type="number"
                disabled={submitting}
                bind:value={formData[field.id]}
              />
            {:else}
              <!-- fallback/default is a normal text input -->
              <input
                id={field.id}
                type="text"
                disabled={submitting}
                bind:value={formData[field.id]}
              />
            {/if}
          </div>
        {/each}

        <div class="actions">
          <button
            type="submit"
            class="btn preset-filled-primary-500"
            disabled={submitting}
          >
            {#if submitting}
              <span class="spinner"></span>
              Uploading...
            {:else}
              Upload
            {/if}
          </button>
          <button
            type="button"
            class="btn preset-filled-surface-500"
            disabled={submitting}
            on:click={handleCancel}>Cancel</button
          >
        </div>
      </form>
    {/if}
  </div>
</div>

<style>
  h1 {
    font-size: 1.5rem;
    margin-bottom: 1.5rem;
    text-align: center;
  }

  .dynamic-form {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    text-align: left;
  }

  label {
    font-weight: 500;
    font-size: 0.9rem;
  }

  input,
  textarea {
    padding: 0.75rem;
    border-radius: 6px;
    border: 1px solid #444;
    background-color: #222;
    color: #fff;
    font-size: 1rem;
    transition: border-color 0.2s;
  }

  input:focus,
  textarea:focus {
    outline: none;
    border-color: #646cff;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 1rem;
    margin-top: 1rem;
  }

  .spinner {
    display: inline-block;
    width: 1rem;
    height: 1rem;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-radius: 50%;
    border-top-color: #fff;
    animation: spin 0.8s linear infinite;
    margin-right: 0.5rem;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>

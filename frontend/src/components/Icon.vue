<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    :stroke="color"
    stroke-width="1.5"
    stroke-linecap="round"
    stroke-linejoin="round"
    class="icon"
    :class="[`icon-${name}`, { spin: spinning }]"
  >
    <!-- Settings / Gear -->
    <template v-if="name === 'settings'">
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
    </template>

    <!-- X / Close -->
    <template v-else-if="name === 'x'">
      <line x1="18" y1="6" x2="6" y2="18" />
      <line x1="6" y1="6" x2="18" y2="18" />
    </template>

    <!-- Brain / Thinking -->
    <template v-else-if="name === 'brain'">
      <path d="M12 2a5 5 0 0 1 4.54 2.88A4.5 4.5 0 0 1 21 9.5a4.5 4.5 0 0 1-2.4 3.98A5 5 0 0 1 12 18" />
      <path d="M12 2a5 5 0 0 0-4.54 2.88A4.5 4.5 0 0 0 3 9.5a4.5 4.5 0 0 0 2.4 3.98A5 5 0 0 0 12 18" />
      <path d="M12 18v4" />
      <path d="M8 10h.01" />
      <path d="M16 10h.01" />
      <path d="M12 14a2 2 0 1 0 0-4 2 2 0 0 0 0 4z" />
    </template>

    <!-- Camera / Screenshot -->
    <template v-else-if="name === 'camera'">
      <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z" />
      <circle cx="12" cy="13" r="4" />
    </template>

    <!-- Mic / Microphone -->
    <template v-else-if="name === 'mic'">
      <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z" />
      <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
      <line x1="12" y1="19" x2="12" y2="23" />
      <line x1="8" y1="23" x2="16" y2="23" />
    </template>

    <!-- Speaker / Volume (system audio) -->
    <template v-else-if="name === 'speaker'">
      <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5" />
      <path d="M15.54 8.46a5 5 0 0 1 0 7.07" />
      <path d="M19.07 4.93a10 10 0 0 1 0 14.14" />
    </template>

    <!-- Copy -->
    <template v-else-if="name === 'copy'">
      <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
    </template>

    <!-- Check -->
    <template v-else-if="name === 'check'">
      <polyline points="20 6 9 17 4 12" />
    </template>

    <!-- Stop / Square (实心方块，用于停止搜题) -->
    <template v-else-if="name === 'square'">
      <rect x="6.5" y="6.5" width="11" height="11" rx="1.5" fill="currentColor" stroke="none" />
    </template>

    <!-- Check Circle -->
    <template v-else-if="name === 'check-circle'">
      <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
      <polyline points="22 4 12 14.01 9 11.01" />
    </template>

    <!-- Alert Triangle / Warning -->
    <template v-else-if="name === 'alert-triangle'">
      <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
      <line x1="12" y1="9" x2="12" y2="13" />
      <line x1="12" y1="17" x2="12.01" y2="17" />
    </template>

    <!-- X Circle / Error -->
    <template v-else-if="name === 'x-circle'">
      <circle cx="12" cy="12" r="10" />
      <line x1="15" y1="9" x2="9" y2="15" />
      <line x1="9" y1="9" x2="15" y2="15" />
    </template>

    <!-- Info -->
    <template v-else-if="name === 'info'">
      <circle cx="12" cy="12" r="10" />
      <line x1="12" y1="16" x2="12" y2="12" />
      <line x1="12" y1="8" x2="12.01" y2="8" />
    </template>

    <!-- Lightbulb -->
    <template v-else-if="name === 'lightbulb'">
      <path d="M9 18h6" />
      <path d="M10 22h4" />
      <path d="M12 2a7 7 0 0 0-4 12.7V17h8v-2.3A7 7 0 0 0 12 2z" />
    </template>

    <!-- File / Document -->
    <template v-else-if="name === 'file'">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <polyline points="14 2 14 8 20 8" />
      <line x1="16" y1="13" x2="8" y2="13" />
      <line x1="16" y1="17" x2="8" y2="17" />
    </template>

    <!-- Trash / Delete -->
    <template v-else-if="name === 'trash'">
      <polyline points="3 6 5 6 21 6" />
      <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
      <line x1="10" y1="11" x2="10" y2="17" />
      <line x1="14" y1="11" x2="14" y2="17" />
    </template>

    <!-- Plus -->
    <template v-else-if="name === 'plus'">
      <line x1="12" y1="5" x2="12" y2="19" />
      <line x1="5" y1="12" x2="19" y2="12" />
    </template>

    <!-- Image / Export -->
    <template v-else-if="name === 'image'">
      <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
      <circle cx="8.5" cy="8.5" r="1.5" />
      <polyline points="21 15 16 10 5 21" />
    </template>

    <!-- Refresh -->
    <template v-else-if="name === 'refresh'">
      <polyline points="23 4 23 10 17 10" />
      <polyline points="1 20 1 14 7 14" />
      <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
    </template>

    <!-- Play -->
    <template v-else-if="name === 'play'">
      <polygon points="5 3 19 12 5 21 5 3" />
    </template>

    <!-- Send -->
    <template v-else-if="name === 'send'">
      <line x1="22" y1="2" x2="11" y2="13" />
      <polygon points="22 2 15 22 11 13 2 9 22 2" />
    </template>

    <!-- Chevron Right -->
    <template v-else-if="name === 'chevron-right'">
      <polyline points="9 18 15 12 9 6" />
    </template>

    <!-- Chevron Down -->
    <template v-else-if="name === 'chevron-down'">
      <polyline points="6 9 12 15 18 9" />
    </template>

    <!-- Chevron Left -->
    <template v-else-if="name === 'chevron-left'">
      <polyline points="15 18 9 12 15 6" />
    </template>

    <!-- Chevron Up -->
    <template v-else-if="name === 'chevron-up'">
      <polyline points="18 15 12 9 6 15" />
    </template>

    <!-- More Vertical (dots) -->
    <template v-else-if="name === 'more-vertical'">
      <circle cx="12" cy="12" r="1" />
      <circle cx="12" cy="5" r="1" />
      <circle cx="12" cy="19" r="1" />
    </template>

    <!-- Clock / Timer -->
    <template v-else-if="name === 'clock'">
      <circle cx="12" cy="12" r="10" />
      <polyline points="12 6 12 12 16 14" />
    </template>

    <!-- Sun (light mode) -->
    <template v-else-if="name === 'sun'">
      <circle cx="12" cy="12" r="5" />
      <line x1="12" y1="1" x2="12" y2="3" />
      <line x1="12" y1="21" x2="12" y2="23" />
      <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
      <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
      <line x1="1" y1="12" x2="3" y2="12" />
      <line x1="21" y1="12" x2="23" y2="12" />
      <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
      <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
    </template>

    <!-- Moon (dark mode) -->
    <template v-else-if="name === 'moon'">
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
    </template>

    <!-- Sidebar collapse (panel-left-close) -->
    <template v-else-if="name === 'panel-left'">
      <rect x="3" y="3" width="18" height="18" rx="2" />
      <line x1="9" y1="3" x2="9" y2="21" />
      <path d="M15 9l-3 3 3 3" />
    </template>

    <!-- Sidebar expand (panel-left-open) -->
    <template v-else-if="name === 'panel-right'">
      <rect x="3" y="3" width="18" height="18" rx="2" />
      <line x1="9" y1="3" x2="9" y2="21" />
      <path d="M14 9l3 3-3 3" />
    </template>

    <!-- Arrow Up -->
    <template v-else-if="name === 'arrow-up'">
      <line x1="12" y1="19" x2="12" y2="5" />
      <polyline points="5 12 12 5 19 12" />
    </template>

    <!-- Arrow Down -->
    <template v-else-if="name === 'arrow-down'">
      <line x1="12" y1="5" x2="12" y2="19" />
      <polyline points="19 12 12 19 5 12" />
    </template>

    <!-- Download -->
    <template v-else-if="name === 'download'">
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
      <polyline points="7 10 12 15 17 10" />
      <line x1="12" y1="15" x2="12" y2="3" />
    </template>

    <!-- Search -->
    <template v-else-if="name === 'search'">
      <circle cx="11" cy="11" r="8" />
      <line x1="21" y1="21" x2="16.65" y2="16.65" />
    </template>

    <!-- ThumbsUp -->
    <template v-else-if="name === 'thumbs-up'">
      <path d="M14 9V5a3 3 0 0 0-3-3l-4 9v11h11.28a2 2 0 0 0 2-1.7l1.38-9a2 2 0 0 0-2-2.3zM7 22H4a2 2 0 0 1-2-2v-7a2 2 0 0 1 2-2h3" />
    </template>

    <!-- ThumbsDown -->
    <template v-else-if="name === 'thumbs-down'">
      <path d="M10 15v4a3 3 0 0 0 3 3l4-9V2H5.72a2 2 0 0 0-2 1.7l-1.38 9a2 2 0 0 0 2 2.3zm7-13h2.67A2.31 2.31 0 0 1 22 4v7a2.31 2.31 0 0 1-2.33 2H17" />
    </template>

    <!-- Sparkles / AI -->
    <template v-else-if="name === 'sparkles'">
      <path d="M12 3l1.912 5.813a2 2 0 0 0 1.275 1.275L21 12l-5.813 1.912a2 2 0 0 0-1.275 1.275L12 21l-1.912-5.813a2 2 0 0 0-1.275-1.275L3 12l5.813-1.912a2 2 0 0 0 1.275-1.275L12 3z" />
    </template>

    <!-- Zap / Speed -->
    <template v-else-if="name === 'zap'">
      <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
    </template>

    <!-- Eye (visible) -->
    <template v-else-if="name === 'eye'">
      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
      <circle cx="12" cy="12" r="3" />
    </template>

    <!-- EyeOff (hidden) -->
    <template v-else-if="name === 'eye-off'">
      <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
      <line x1="1" y1="1" x2="23" y2="23" />
    </template>

    <!-- Mouse pointer (click through) -->
    <template v-else-if="name === 'mouse-pointer'">
      <path d="M3 3l7.07 16.97 2.51-7.39 7.39-2.51L3 3z" />
      <path d="M13 13l6 6" />
    </template>

    <!-- Move -->
    <template v-else-if="name === 'move'">
      <polyline points="5 9 2 12 5 15" />
      <polyline points="9 5 12 2 15 5" />
      <polyline points="15 19 12 22 9 19" />
      <polyline points="19 9 22 12 19 15" />
      <line x1="2" y1="12" x2="22" y2="12" />
      <line x1="12" y1="2" x2="12" y2="22" />
    </template>

    <!-- Power / Quit -->
    <template v-else-if="name === 'power'">
      <path d="M18.36 6.64a9 9 0 1 1-12.73 0" />
      <line x1="12" y1="2" x2="12" y2="12" />
    </template>

    <!-- Code -->
    <template v-else-if="name === 'code'">
      <polyline points="16 18 22 12 16 6" />
      <polyline points="8 6 2 12 8 18" />
    </template>

    <!-- Wrap / text wrapping -->
    <template v-else-if="name === 'wrap'">
      <line x1="3" y1="6" x2="21" y2="6" />
      <path d="M3 12h15a3 3 0 1 1 0 6h-4" />
      <polyline points="16 16 14 18 16 20" />
      <line x1="3" y1="18" x2="10" y2="18" />
    </template>

    <!-- External link -->
    <template v-else-if="name === 'external-link'">
      <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
      <polyline points="15 3 21 3 21 9" />
      <line x1="10" y1="14" x2="21" y2="3" />
    </template>

    <!-- Loader / Spinner -->
    <template v-else-if="name === 'loader'">
      <line x1="12" y1="2" x2="12" y2="6" />
      <line x1="12" y1="18" x2="12" y2="22" />
      <line x1="4.93" y1="4.93" x2="7.76" y2="7.76" />
      <line x1="16.24" y1="16.24" x2="19.07" y2="19.07" />
      <line x1="2" y1="12" x2="6" y2="12" />
      <line x1="18" y1="12" x2="22" y2="12" />
      <line x1="4.93" y1="19.07" x2="7.76" y2="16.24" />
      <line x1="16.24" y1="7.76" x2="19.07" y2="4.93" />
    </template>

    <!-- Key / Keyboard -->
    <template v-else-if="name === 'keyboard'">
      <rect x="2" y="4" width="20" height="16" rx="2" />
      <line x1="6" y1="8" x2="6" y2="8" />
      <line x1="10" y1="8" x2="10" y2="8" />
      <line x1="14" y1="8" x2="14" y2="8" />
      <line x1="18" y1="8" x2="18" y2="8" />
      <line x1="6" y1="12" x2="6" y2="12" />
      <line x1="10" y1="12" x2="10" y2="12" />
      <line x1="14" y1="12" x2="14" y2="12" />
      <line x1="18" y1="12" x2="18" y2="12" />
      <line x1="7" y1="16" x2="17" y2="16" />
    </template>

    <!-- footprints (lucide) -->
    <template v-else-if="name === 'footprints'">
      <path d="M4 16v-2.38C4 11.5 2.97 10.5 3 8c.03-2.72 1.49-6 4.5-6C9.37 2 10 3.8 10 5.5c0 3.11-2 5.66-2 8.68V16a2 2 0 1 1-4 0Z" /> <path d="M20 20v-2.38c0-2.12 1.03-3.12 1-5.62-.03-2.72-1.49-6-4.5-6C14.63 6 14 7.8 14 9.5c0 3.11 2 5.66 2 8.68V20a2 2 0 1 0 4 0Z" /> <path d="M16 17h4" /> <path d="M4 13h4" />
    </template>

    <!-- book-open (lucide) -->
    <template v-else-if="name === 'book-open'">
      <path d="M12 7v14" /> <path d="M3 18a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h5a4 4 0 0 1 4 4 4 4 0 0 1 4-4h5a1 1 0 0 1 1 1v13a1 1 0 0 1-1 1h-6a3 3 0 0 0-3 3 3 3 0 0 0-3-3z" />
    </template>

    <!-- languages (lucide) -->
    <template v-else-if="name === 'languages'">
      <path d="m5 8 6 6" /> <path d="m4 14 6-6 2-3" /> <path d="M2 5h12" /> <path d="M7 2h1" /> <path d="m22 22-5-10-5 10" /> <path d="M14 18h6" />
    </template>

    <!-- pen-line (lucide) -->
    <template v-else-if="name === 'pen-line'">
      <path d="M12 20h9" /> <path d="M16.376 3.622a1 1 0 0 1 3.002 3.002L7.368 18.635a2 2 0 0 1-.855.506l-2.872.838a.5.5 0 0 1-.62-.62l.838-2.872a2 2 0 0 1 .506-.854z" />
    </template>

    <!-- glasses (lucide) -->
    <template v-else-if="name === 'glasses'">
      <circle cx="6" cy="15" r="4" /> <circle cx="18" cy="15" r="4" /> <path d="M14 15a2 2 0 0 0-2-2 2 2 0 0 0-2 2" /> <path d="M2.5 13 5 7c.7-1.3 1.4-2 3-2" /> <path d="M21.5 13 19 7c-.7-1.3-1.5-2-3-2" />
    </template>

    <!-- coffee (lucide) -->
    <template v-else-if="name === 'coffee'">
      <path d="M10 2v2" /> <path d="M14 2v2" /> <path d="M16 8a1 1 0 0 1 1 1v8a4 4 0 0 1-4 4H7a4 4 0 0 1-4-4V9a1 1 0 0 1 1-1h14a4 4 0 1 1 0 8h-1" /> <path d="M6 2v2" />
    </template>

    <!-- terminal (lucide) -->
    <template v-else-if="name === 'terminal'">
      <polyline points="4 17 10 11 4 5" /> <line x1="12" x2="20" y1="19" y2="19" />
    </template>

    <!-- cpu (lucide) -->
    <template v-else-if="name === 'cpu'">
      <rect width="16" height="16" x="4" y="4" rx="2" /> <rect width="6" height="6" x="9" y="9" rx="1" /> <path d="M15 2v2" /> <path d="M15 20v2" /> <path d="M2 15h2" /> <path d="M2 9h2" /> <path d="M20 15h2" /> <path d="M20 9h2" /> <path d="M9 2v2" /> <path d="M9 20v2" />
    </template>

    <!-- atom (lucide) -->
    <template v-else-if="name === 'atom'">
      <circle cx="12" cy="12" r="1" /> <path d="M20.2 20.2c2.04-2.03.02-7.36-4.5-11.9-4.54-4.52-9.87-6.54-11.9-4.5-2.04 2.03-.02 7.36 4.5 11.9 4.54 4.52 9.87 6.54 11.9 4.5Z" /> <path d="M15.7 15.7c4.52-4.54 6.54-9.87 4.5-11.9-2.03-2.04-7.36-.02-11.9 4.5-4.52 4.54-6.54 9.87-4.5 11.9 2.03 2.04 7.36.02 11.9-4.5Z" />
    </template>

    <!-- braces (lucide) -->
    <template v-else-if="name === 'braces'">
      <path d="M8 3H7a2 2 0 0 0-2 2v5a2 2 0 0 1-2 2 2 2 0 0 1 2 2v5c0 1.1.9 2 2 2h1" /> <path d="M16 21h1a2 2 0 0 0 2-2v-5c0-1.1.9-2 2-2a2 2 0 0 1-2-2V5a2 2 0 0 0-2-2h-1" />
    </template>

    <!-- database (lucide) -->
    <template v-else-if="name === 'database'">
      <ellipse cx="12" cy="5" rx="9" ry="3" /> <path d="M3 5V19A9 3 0 0 0 21 19V5" /> <path d="M3 12A9 3 0 0 0 21 12" />
    </template>

    <!-- ship (lucide) -->
    <template v-else-if="name === 'ship'">
      <path d="M12 10.189V14" /> <path d="M12 2v3" /> <path d="M19 13V7a2 2 0 0 0-2-2H7a2 2 0 0 0-2 2v6" /> <path d="M19.38 20A11.6 11.6 0 0 0 21 14l-8.188-3.639a2 2 0 0 0-1.624 0L3 14a11.6 11.6 0 0 0 2.81 7.76" /> <path d="M2 21c.6.5 1.2 1 2.5 1 2.5 0 2.5-2 5-2 1.3 0 1.9.5 2.5 1s1.2 1 2.5 1c2.5 0 2.5-2 5-2 1.3 0 1.9.5 2.5 1" />
    </template>

    <!-- shield (lucide) -->
    <template v-else-if="name === 'shield'">
      <path d="M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z" />
    </template>

    <!-- binary (lucide) -->
    <template v-else-if="name === 'binary'">
      <rect x="14" y="14" width="4" height="6" rx="2" /> <rect x="6" y="4" width="4" height="6" rx="2" /> <path d="M6 20h4" /> <path d="M14 10h4" /> <path d="M6 14h2v6" /> <path d="M14 4h2v6" />
    </template>

    <!-- calendar (lucide) -->
    <template v-else-if="name === 'calendar'">
      <path d="M8 2v4" /> <path d="M16 2v4" /> <rect width="18" height="18" x="3" y="4" rx="2" /> <path d="M3 10h18" />
    </template>

    <!-- palette (lucide) -->
    <template v-else-if="name === 'palette'">
      <circle cx="13.5" cy="6.5" r=".5" fill="currentColor" /> <circle cx="17.5" cy="10.5" r=".5" fill="currentColor" /> <circle cx="8.5" cy="7.5" r=".5" fill="currentColor" /> <circle cx="6.5" cy="12.5" r=".5" fill="currentColor" /> <path d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10c.926 0 1.648-.746 1.648-1.688 0-.437-.18-.835-.437-1.125-.29-.289-.438-.652-.438-1.125a1.64 1.64 0 0 1 1.668-1.668h1.996c3.051 0 5.555-2.503 5.555-5.554C21.965 6.012 17.461 2 12 2z" />
    </template>

    <!-- mouse (lucide) -->
    <template v-else-if="name === 'mouse'">
      <rect x="5" y="2" width="14" height="20" rx="7" /> <path d="M12 6v4" />
    </template>

    <!-- building-2 (lucide) -->
    <template v-else-if="name === 'building-2'">
      <path d="M6 22V4a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v18Z" /> <path d="M6 12H4a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2h2" /> <path d="M18 9h2a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-2" /> <path d="M10 6h4" /> <path d="M10 10h4" /> <path d="M10 14h4" /> <path d="M10 18h4" />
    </template>

    <!-- gamepad-2 (lucide) -->
    <template v-else-if="name === 'gamepad-2'">
      <line x1="6" x2="10" y1="11" y2="11" /> <line x1="8" x2="8" y1="9" y2="13" /> <line x1="15" x2="15.01" y1="12" y2="12" /> <line x1="18" x2="18.01" y1="10" y2="10" /> <path d="M17.32 5H6.68a4 4 0 0 0-3.978 3.59c-.006.052-.01.101-.017.152C2.604 9.416 2 14.456 2 16a3 3 0 0 0 3 3c1 0 1.5-.5 2-1l1.414-1.414A2 2 0 0 1 9.828 16h4.344a2 2 0 0 1 1.414.586L17 18c.5.5 1 1 2 1a3 3 0 0 0 3-3c0-1.545-.604-6.584-.685-7.258-.007-.05-.011-.1-.017-.151A4 4 0 0 0 17.32 5z" />
    </template>

    <!-- clipboard-list (lucide) -->
    <template v-else-if="name === 'clipboard-list'">
      <rect width="8" height="4" x="8" y="2" rx="1" ry="1" /> <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2" /> <path d="M12 11h4" /> <path d="M12 16h4" /> <path d="M8 11h.01" /> <path d="M8 16h.01" />
    </template>

    <!-- map (lucide) -->
    <template v-else-if="name === 'map'">
      <path d="M14.106 5.553a2 2 0 0 0 1.788 0l3.659-1.83A1 1 0 0 1 21 4.619v12.764a1 1 0 0 1-.553.894l-4.553 2.277a2 2 0 0 1-1.788 0l-4.212-2.106a2 2 0 0 0-1.788 0l-3.659 1.83A1 1 0 0 1 3 19.381V6.618a1 1 0 0 1 .553-.894l4.553-2.277a2 2 0 0 1 1.788 0z" /> <path d="M15 5.764v15" /> <path d="M9 3.236v15" />
    </template>

    <!-- puzzle (lucide) -->
    <template v-else-if="name === 'puzzle'">
      <path d="M15.39 4.39a1 1 0 0 0 1.68-.474 2.5 2.5 0 1 1 3.014 3.015 1 1 0 0 0-.474 1.68l1.683 1.682a2.414 2.414 0 0 1 0 3.414L19.61 15.39a1 1 0 0 1-1.68-.474 2.5 2.5 0 1 0-3.014 3.015 1 1 0 0 1 .474 1.68l-1.683 1.682a2.414 2.414 0 0 1-3.414 0L8.61 19.61a1 1 0 0 0-1.68.474 2.5 2.5 0 1 1-3.014-3.015 1 1 0 0 0 .474-1.68l-1.683-1.682a2.414 2.414 0 0 1 0-3.414L4.39 8.61a1 1 0 0 1 1.68.474 2.5 2.5 0 1 0 3.014-3.015 1 1 0 0 1-.474-1.68l1.683-1.682a2.414 2.414 0 0 1 3.414 0z" />
    </template>

    <!-- trending-up (lucide) -->
    <template v-else-if="name === 'trending-up'">
      <polyline points="22 7 13.5 15.5 8.5 10.5 2 17" /> <polyline points="16 7 22 7 22 13" />
    </template>

    <!-- bug (lucide) -->
    <template v-else-if="name === 'bug'">
      <path d="m8 2 1.88 1.88" /> <path d="M14.12 3.88 16 2" /> <path d="M9 7.13v-1a3.003 3.003 0 1 1 6 0v1" /> <path d="M12 20c-3.3 0-6-2.7-6-6v-3a4 4 0 0 1 4-4h4a4 4 0 0 1 4 4v3c0 3.3-2.7 6-6 6" /> <path d="M12 20v-9" /> <path d="M6.53 9C4.6 8.8 3 7.1 3 5" /> <path d="M6 13H2" /> <path d="M3 21c0-2.1 1.7-3.9 3.8-4" /> <path d="M20.97 5c0 2.1-1.6 3.8-3.5 4" /> <path d="M22 13h-4" /> <path d="M17.2 17c2.1.1 3.8 1.9 3.8 4" />
    </template>

    <!-- file-text (lucide) -->
    <template v-else-if="name === 'file-text'">
      <path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z" /> <path d="M14 2v4a2 2 0 0 0 2 2h4" /> <path d="M10 9H8" /> <path d="M16 13H8" /> <path d="M16 17H8" />
    </template>

    <!-- message-square (lucide) -->
    <template v-else-if="name === 'message-square'">
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
    </template>

    <!-- gem (lucide) -->
    <template v-else-if="name === 'gem'">
      <path d="M6 3h12l4 6-10 13L2 9Z" /> <path d="M11 3 8 9l4 13 4-13-3-6" /> <path d="M2 9h20" />
    </template>

    <!-- smartphone (lucide) -->
    <template v-else-if="name === 'smartphone'">
      <rect width="14" height="20" x="5" y="2" rx="2" ry="2" /> <path d="M12 18h.01" />
    </template>

    <!-- party-popper (lucide) -->
    <template v-else-if="name === 'party-popper'">
      <path d="M5.8 11.3 2 22l10.7-3.79" /> <path d="M4 3h.01" /> <path d="M22 8h.01" /> <path d="M15 2h.01" /> <path d="M22 20h.01" /> <path d="m22 2-2.24.75a2.9 2.9 0 0 0-1.96 3.12c.1.86-.57 1.63-1.45 1.63h-.38c-.86 0-1.6.6-1.76 1.44L14 10" /> <path d="m22 13-.82-.33c-.86-.34-1.82.2-1.98 1.11c-.11.7-.72 1.22-1.43 1.22H17" /> <path d="m11 2 .33.82c.34.86-.2 1.82-1.11 1.98C9.52 4.9 9 5.52 9 6.23V7" /> <path d="M11 13c1.93 1.93 2.83 4.17 2 5-.83.83-3.07-.07-5-2-1.93-1.93-2.83-4.17-2-5 .83-.83 3.07.07 5 2Z" />
    </template>

    <!-- clapperboard (lucide) -->
    <template v-else-if="name === 'clapperboard'">
      <path d="M20.2 6 3 11l-.9-2.4c-.3-1.1.3-2.2 1.3-2.5l13.5-4c1.1-.3 2.2.3 2.5 1.3Z" /> <path d="m6.2 5.3 3.1 3.9" /> <path d="m12.4 3.4 3.1 4" /> <path d="M3 11h18v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z" />
    </template>

    <!-- divide (lucide) -->
    <template v-else-if="name === 'divide'">
      <circle cx="12" cy="6" r="1" /> <line x1="5" x2="19" y1="12" y2="12" /> <circle cx="12" cy="18" r="1" />
    </template>

    <!-- flag (lucide) -->
    <template v-else-if="name === 'flag'">
      <path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z" /> <line x1="4" x2="4" y1="22" y2="15" />
    </template>

    <!-- globe (lucide) -->
    <template v-else-if="name === 'globe'">
      <circle cx="12" cy="12" r="10" /> <path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20" /> <path d="M2 12h20" />
    </template>

    <!-- scroll-text (lucide) -->
    <template v-else-if="name === 'scroll-text'">
      <path d="M15 12h-5" /> <path d="M15 8h-5" /> <path d="M19 17V5a2 2 0 0 0-2-2H4" /> <path d="M8 21h12a2 2 0 0 0 2-2v-1a1 1 0 0 0-1-1H11a1 1 0 0 0-1 1v1a2 2 0 1 1-4 0V5a2 2 0 1 0-4 0v2a1 1 0 0 0 1 1h3" />
    </template>

    <!-- orbit (lucide) -->
    <template v-else-if="name === 'orbit'">
      <circle cx="12" cy="12" r="3" /> <circle cx="19" cy="5" r="2" /> <circle cx="5" cy="19" r="2" /> <path d="M10.4 21.9a10 10 0 0 0 9.941-15.416" /> <path d="M13.5 2.1a10 10 0 0 0-9.841 15.416" />
    </template>

    <!-- flask-conical (lucide) -->
    <template v-else-if="name === 'flask-conical'">
      <path d="M14 2v6a2 2 0 0 0 .245.96l5.51 10.08A2 2 0 0 1 18 22H6a2 2 0 0 1-1.755-2.96l5.51-10.08A2 2 0 0 0 10 8V2" /> <path d="M6.453 15h11.094" /> <path d="M8.5 2h7" />
    </template>

    <!-- briefcase (lucide) -->
    <template v-else-if="name === 'briefcase'">
      <path d="M16 20V4a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16" /> <rect width="20" height="14" x="2" y="6" rx="2" />
    </template>

    <!-- circle-dollar-sign (lucide) -->
    <template v-else-if="name === 'circle-dollar-sign'">
      <circle cx="12" cy="12" r="10" /> <path d="M16 8h-6a2 2 0 1 0 0 4h4a2 2 0 1 1 0 4H8" /> <path d="M12 18V6" />
    </template>

    <!-- receipt (lucide) -->
    <template v-else-if="name === 'receipt'">
      <path d="M4 2v20l2-1 2 1 2-1 2 1 2-1 2 1 2-1 2 1V2l-2 1-2-1-2 1-2-1-2 1-2-1-2 1Z" /> <path d="M16 8h-6a2 2 0 1 0 0 4h4a2 2 0 1 1 0 4H8" /> <path d="M12 17.5v-11" />
    </template>

    <!-- handshake (lucide) -->
    <template v-else-if="name === 'handshake'">
      <path d="m11 17 2 2a1 1 0 1 0 3-3" /> <path d="m14 14 2.5 2.5a1 1 0 1 0 3-3l-3.88-3.88a3 3 0 0 0-4.24 0l-.88.88a1 1 0 1 1-3-3l2.81-2.81a5.79 5.79 0 0 1 7.06-.87l.47.28a2 2 0 0 0 1.42.25L21 4" /> <path d="m21 3 1 11h-2" /> <path d="M3 3 2 14l6.5 6.5a1 1 0 1 0 3-3" /> <path d="M3 4h8" />
    </template>

    <!-- scale (lucide) -->
    <template v-else-if="name === 'scale'">
      <path d="m16 16 3-8 3 8c-.87.65-1.92 1-3 1s-2.13-.35-3-1Z" /> <path d="m2 16 3-8 3 8c-.87.65-1.92 1-3 1s-2.13-.35-3-1Z" /> <path d="M7 21h10" /> <path d="M12 3v18" /> <path d="M3 7h2c2 0 5-1 7-2 2 1 5 2 7 2h2" />
    </template>

    <!-- package (lucide) -->
    <template v-else-if="name === 'package'">
      <path d="M11 21.73a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73z" /> <path d="M12 22V12" /> <path d="m3.3 7 7.703 4.734a2 2 0 0 0 1.994 0L20.7 7" /> <path d="m7.5 4.27 9 5.15" />
    </template>

    <!-- trending-down (lucide) -->
    <template v-else-if="name === 'trending-down'">
      <polyline points="22 17 13.5 8.5 8.5 13.5 2 7" /> <polyline points="16 17 22 17 22 11" />
    </template>

    <!-- rocket (lucide) -->
    <template v-else-if="name === 'rocket'">
      <path d="M4.5 16.5c-1.5 1.26-2 5-2 5s3.74-.5 5-2c.71-.84.7-2.13-.09-2.91a2.18 2.18 0 0 0-2.91-.09z" /> <path d="m12 15-3-3a22 22 0 0 1 2-3.95A12.88 12.88 0 0 1 22 2c0 2.72-.78 7.5-6 11a22.35 22.35 0 0 1-4 2z" /> <path d="M9 12H4s.55-3.03 2-4c1.62-1.08 5 0 5 0" /> <path d="M12 15v5s3.03-.55 4-2c1.08-1.62 0-5 0-5" />
    </template>

    <!-- microscope (lucide) -->
    <template v-else-if="name === 'microscope'">
      <path d="M6 18h8" /> <path d="M3 22h18" /> <path d="M14 22a7 7 0 1 0 0-14h-1" /> <path d="M9 14h2" /> <path d="M9 12a2 2 0 0 1-2-2V6h6v4a2 2 0 0 1-2 2Z" /> <path d="M12 6V3a1 1 0 0 0-1-1H9a1 1 0 0 0-1 1v3" />
    </template>

    <!-- crown (lucide) -->
    <template v-else-if="name === 'crown'">
      <path d="M11.562 3.266a.5.5 0 0 1 .876 0L15.39 8.87a1 1 0 0 0 1.516.294L21.183 5.5a.5.5 0 0 1 .798.519l-2.834 10.246a1 1 0 0 1-.956.734H5.81a1 1 0 0 1-.957-.734L2.02 6.02a.5.5 0 0 1 .798-.519l4.276 3.664a1 1 0 0 0 1.516-.294z" /> <path d="M5 21h14" />
    </template>

    <!-- megaphone (lucide) -->
    <template v-else-if="name === 'megaphone'">
      <path d="m3 11 18-5v12L3 14v-3z" /> <path d="M11.6 16.8a3 3 0 1 1-5.8-1.6" />
    </template>

    <!-- wand-sparkles (lucide) -->
    <template v-else-if="name === 'wand-sparkles'">
      <path d="m21.64 3.64-1.28-1.28a1.21 1.21 0 0 0-1.72 0L2.36 18.64a1.21 1.21 0 0 0 0 1.72l1.28 1.28a1.2 1.2 0 0 0 1.72 0L21.64 5.36a1.2 1.2 0 0 0 0-1.72" /> <path d="m14 7 3 3" /> <path d="M5 6v4" /> <path d="M19 14v4" /> <path d="M10 2v2" /> <path d="M7 8H3" /> <path d="M21 16h-4" /> <path d="M11 3H9" />
    </template>

    <!-- leaf (lucide) -->
    <template v-else-if="name === 'leaf'">
      <path d="M11 20A7 7 0 0 1 9.8 6.1C15.5 5 17 4.48 19 2c1 2 2 4.18 2 8 0 5.5-4.78 10-10 10Z" /> <path d="M2 21c0-3 1.85-5.36 5.08-6C9.5 14.52 12 13 13 12" />
    </template>

    <!-- star (lucide) -->
    <template v-else-if="name === 'star'">
      <path d="M11.525 2.295a.53.53 0 0 1 .95 0l2.31 4.679a2.123 2.123 0 0 0 1.595 1.16l5.166.756a.53.53 0 0 1 .294.904l-3.736 3.638a2.123 2.123 0 0 0-.611 1.878l.882 5.14a.53.53 0 0 1-.771.56l-4.618-2.428a2.122 2.122 0 0 0-1.973 0L6.396 21.01a.53.53 0 0 1-.77-.56l.881-5.139a2.122 2.122 0 0 0-.611-1.879L2.16 9.795a.53.53 0 0 1 .294-.906l5.165-.755a2.122 2.122 0 0 0 1.597-1.16z" />
    </template>

    <!-- table (lucide) -->
    <template v-else-if="name === 'table'">
      <path d="M12 3v18" /> <rect width="18" height="18" x="3" y="3" rx="2" /> <path d="M3 9h18" /> <path d="M3 15h18" />
    </template>

    <!-- shield-check (lucide) -->
    <template v-else-if="name === 'shield-check'">
      <path d="M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z" /> <path d="m9 12 2 2 4-4" />
    </template>

    <!-- briefcase-business (lucide) -->
    <template v-else-if="name === 'briefcase-business'">
      <path d="M12 12h.01" /> <path d="M16 6V4a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2" /> <path d="M22 13a18.15 18.15 0 0 1-20 0" /> <rect width="20" height="14" x="2" y="6" rx="2" />
    </template>

    <!-- heart-handshake (lucide) -->
    <template v-else-if="name === 'heart-handshake'">
      <path d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z" /> <path d="M12 5 9.04 7.96a2.17 2.17 0 0 0 0 3.08c.82.82 2.13.85 3 .07l2.07-1.9a2.82 2.82 0 0 1 3.79 0l2.96 2.66" /> <path d="m18 15-2-2" /> <path d="m15 18-2-2" />
    </template>

    <!-- wrench (lucide) -->
    <template v-else-if="name === 'wrench'">
      <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" />
    </template>

    <!-- brush (lucide) -->
    <template v-else-if="name === 'brush'">
      <path d="m9.06 11.9 8.07-8.06a2.85 2.85 0 1 1 4.03 4.03l-8.06 8.08" /> <path d="M7.07 14.94c-1.66 0-3 1.35-3 3.02 0 1.33-2.5 1.52-2 2.02 1.08 1.1 2.49 2.02 4 2.02 2.2 0 4-1.8 4-4.04a3.01 3.01 0 0 0-3-3.02z" />
    </template>

    <!-- gauge (lucide) -->
    <template v-else-if="name === 'gauge'">
      <path d="m12 14 4-4" /> <path d="M3.34 19a10 10 0 1 1 17.32 0" />
    </template>

    <!-- drama (lucide) -->
    <template v-else-if="name === 'drama'">
      <path d="M10 11h.01" /> <path d="M14 6h.01" /> <path d="M18 6h.01" /> <path d="M6.5 13.1h.01" /> <path d="M22 5c0 9-4 12-6 12s-6-3-6-12c0-2 2-3 6-3s6 1 6 3" /> <path d="M17.4 9.9c-.8.8-2 .8-2.8 0" /> <path d="M10.1 7.1C9 7.2 7.7 7.7 6 8.6c-3.5 2-4.7 3.9-3.7 5.6 4.5 7.8 9.5 8.4 11.2 7.4.9-.5 1.9-2.1 1.9-4.7" /> <path d="M9.1 16.5c.3-1.1 1.4-1.7 2.4-1.4" />
    </template>

    <!-- crop (lucide) -->
    <template v-else-if="name === 'crop'">
      <path d="M6 2v14a2 2 0 0 0 2 2h14" /> <path d="M18 22V8a2 2 0 0 0-2-2H2" />
    </template>

    <!-- monitor (lucide) -->
    <template v-else-if="name === 'monitor'">
      <rect width="20" height="14" x="2" y="3" rx="2" /> <line x1="8" x2="16" y1="21" y2="21" /> <line x1="12" x2="12" y1="17" y2="21" />
    </template>

    <!-- user (lucide) -->
    <template v-else-if="name === 'user'">
      <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" /> <circle cx="12" cy="7" r="4" />
    </template>

    <!-- bot (lucide) -->
    <template v-else-if="name === 'bot'">
      <path d="M12 8V4H8" /> <rect width="16" height="12" x="4" y="8" rx="2" /> <path d="M2 14h2" /> <path d="M20 14h2" /> <path d="M15 13v2" /> <path d="M9 13v2" />
    </template>

    <!-- bar-chart-3 (lucide: chart-bar) -->
    <template v-else-if="name === 'bar-chart-3'">
      <path d="M3 3v16a2 2 0 0 0 2 2h16" /> <path d="M7 16h8" /> <path d="M7 11h12" /> <path d="M7 6h3" />
    </template>

    <!-- pie-chart (lucide) -->
    <template v-else-if="name === 'pie-chart'">
      <path d="M21 12c.552 0 1.005-.449.95-.998a10 10 0 0 0-8.953-8.951c-.55-.055-.998.398-.998.95v8a1 1 0 0 0 1 1z" /> <path d="M21.21 15.89A10 10 0 1 1 8 2.83" />
    </template>

    <!-- Fallback: question mark -->
    <template v-else>
      <circle cx="12" cy="12" r="10" />
      <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
      <line x1="12" y1="17" x2="12.01" y2="17" />
    </template>
  </svg>
</template>

<script setup>
defineProps({
  name: { type: String, required: true },
  size: { type: [Number, String], default: 18 },
  color: { type: String, default: 'currentColor' },
  spinning: { type: Boolean, default: false }
})
</script>

<style scoped>
.icon {
  display: inline-flex;
  flex-shrink: 0;
  vertical-align: middle;
}
.icon.spin {
  animation: iconSpin 0.8s linear infinite;
}
@keyframes iconSpin {
  to { transform: rotate(360deg); }
}
</style>

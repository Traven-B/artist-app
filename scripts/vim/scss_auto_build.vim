function! s:FindBuildScript(startdir) abort
  let l:dir = a:startdir
  let l:home = expand('~')
  while 1
    " 1. bin/build-scss.sh (your organized dir, most likely)
    let l:binscript = l:dir . '/bin/build-scss.sh'
    if filereadable(l:binscript)
      return l:binscript
    endif

    " 2. .build-scss.sh (hidden standalone)
    let l:hidden = l:dir . '/.build-scss.sh'
    if filereadable(l:hidden)
      return l:hidden
    endif

    " 3. build-scss.sh (visible standalone)
    let l:standalone = l:dir . '/build-scss.sh'
    if filereadable(l:standalone)
      return l:standalone
    endif

    if l:dir ==# l:home || l:dir ==# '/'
      return ''
    endif
    let l:dir = fnamemodify(l:dir, ':h')
  endwhile
endfunction

function! BuildScssForCurrentFile() abort
  " Only act on .scss files
  if expand('%:e') !=# 'scss'
    return
  endif

  let l:startdir = expand('%:p:h')
  let l:script = <SID>FindBuildScript(l:startdir)
  if empty(l:script)
    echo "No bin/build-scss.sh found up to $HOME"
    return
  endif

  " Run the script, capturing both stdout and stderr
  let l:cmd = shellescape(l:script)
  let l:output = system(l:cmd . ' 2>&1')
  let l:status = v:shell_error

  " Success: just a transient message, no pane
  " if l:status == 0
  "   echom "SCSS build OK"
  "   return
  " endif

if l:status == 0
  " Close error window entirely on success
  let l:bufnr = bufnr('__scss_build__')
  if l:bufnr != -1
    let l:winid = bufwinid(l:bufnr)
    if l:winid != -1
      let l:curwinid = win_getid()  " Save current window
      call win_gotoid(l:winid)      " Go to error window
      q!                           " Close it completely
      call win_gotoid(l:curwinid)   " Back to SCSS file
    endif
    execute 'bwipeout ' . l:bufnr  " Wipe the buffer too
  endif
  echom "SCSS build OK"
  return
endif

  " Failure: show / reuse the build window using window IDs
  let l:bufname = '__scss_build__'
  let l:bufnr = bufnr(l:bufname)
  let l:winid = bufwinid(l:bufnr)  " Get window ID for this buffer (-1 if none)

  if l:winid == -1
    " No window: create new split
    botright 10split  " Fixed height
    enew
    let l:bufnr = bufnr('%')
    execute 'file ' . l:bufname
    setlocal buftype=nofile bufhidden=wipe nobuflisted noswapfile
    setlocal nowrap
  else
    " Reuse existing window, preserve cursor position
    let l:curwinid = win_getid()  " Current window ID before switching
    call win_gotoid(l:winid)      " Switch to build window
  endif

  " Update buffer content (safe now)
  call deletebufline(l:bufnr, 1, '$')
  call setbufline(l:bufnr, 1, split(l:output, "\n"))
  execute 1
  normal! gg

  " Return focus to original window
  if exists('l:curwinid')
    call win_gotoid(l:curwinid)
  endif

  echom "SCSS build FAILED (see __scss_build__)"
endfunction

command! ScssBuildOn  let g:scss_auto_build = 1 | echom "SCSS auto-build ON"
command! ScssBuildOff let g:scss_auto_build = 0 | echom "SCSS auto-build OFF"
command! ScssBuildToggle let g:scss_auto_build = !get(g:, 'scss_auto_build', 1) | echom "SCSS auto-build " . (g:scss_auto_build ? "ON" : "OFF")

" Default to ON
if !exists('g:scss_auto_build')
  let g:scss_auto_build = 1
endif

augroup ScssAutoBuild
  autocmd!
  autocmd BufWritePost *.scss if g:scss_auto_build | call BuildScssForCurrentFile() | endif
augroup END

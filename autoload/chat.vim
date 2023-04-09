fun s:OnClose(winid)
    let info = getwininfo(a:winid)

    if len(info) == 0
        " Window doesn't exist
        return
    endif

    let job = get(info[0].variables, 'chat_job', '')

    if job == ''
        return
    endif

    call chat#stop(job)
endfun

au WinClosed * call <SID>OnClose(win_getid())

fun chat#new()
    88vsplit
    noswapfile hide enew
    setlocal buftype=nofile
    setlocal bufhidden=hide
    setlocal filetype=chatgpt.markdown
    setlocal nospell
    " Don't automatically insert the comments leader when pressing enter (r)
    " setlocal formatoptions-=r
    setlocal formatoptions=tc
    setlocal comments=b:>
    " Clear all indent keys to prevent them from triggering a re-indentation at
    " unexpected times, particularly in code, with the exception of Enter
    setlocal indentkeys=o
    setlocal indentexpr=chat#indentexpr()
    setlocal winfixwidth

    silent file Chat

    let outbuf = bufnr()
    let outwin = win_getid()
    let chat_job = chat#start(outbuf)

    let w:outwin = outwin
    let w:chat_job = chat_job

    10split
    noswapfile hide enew
    setlocal buftype=nofile
    setlocal bufhidden=hide
    setlocal filetype=markdown
    setlocal winfixheight

    silent file Input

    nmap <buffer> <CR> <Cmd>call chat#sendbuf()<CR>
    let w:outwin = outwin
    let w:chat_job = chat_job
endfun

fun chat#indentexpr()
    " TODO: Get rid of this hack somehow
    if g:IsMkdCode(v:lnum)
        return 0
    else
        return GetMarkdownIndent()
    endif
endfun

fun chat#start(outbuf)
    return job_start([g:vim_chatgpt_binary], #{
    \  mode: 'raw',
    \  in_io: 'pipe',
    \  out_io: 'pipe',
    \  err_io: 'pipe',
    \  out_cb: 'chat#output',
    \  err_cb: 'chat#error',
    \  exit_cb: 'chat#exit',
    \})
endfun

fun chat#stop(job)
    if job_status(a:job) == "run"
        call job_stop(a:job)
        echom "Stopped"
    endif
endfun

fun chat#send(msg)
    if exists('w:chat_job') && job_status(w:chat_job) == "run"
        call ch_sendraw(w:chat_job, json_encode(#{text: a:msg}).."\n")
    endif
endfun

fun chat#sendbuf()
    let md = py3eval('mdjoin.join()')
    %d
    " call win_execute(w:outwin, 'normal ggzb', 1)
    call chat#send(md)
endfun

fun chat#output(job, text)
    let prevreg = @"
    let @" = a:text
    " call win_execute(w:outwin, 'exec "normal" "Gzb$a\<C-r>\""', 1)
    call win_execute(w:outwin, 'call chat#dumptext()', 1)
    let @" = prevreg
endfun

fun chat#dumptext()
    " TODO: Get rid of this hack somehow
    if g:IsMkdCode(line('.'))
        setlocal formatoptions-=t
    else
        setlocal formatoptions+=t
    endif

    exec "normal!" "Gzb$a\<C-r>\""
endfun

fun chat#exit(job, status)
    for winfo in getwininfo()
        if get(winfo.variables, 'chat_job', '') == a:job
            call win_execute(winfo.winid, 'q', 1)
        endif
    endfor

    if a:status >= 0
        echom "Exit:" a:status
    endif
endfun

fun chat#error(job, msg)
    echohl Error
    echom "[Chat Error]" a:msg
    echohl None
endfun

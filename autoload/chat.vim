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

fun chat#new(model='')
    88vsplit
    noswapfile hide enew
    setlocal buftype=nofile
    setlocal bufhidden=wipe
    setlocal nobuflisted
    setlocal filetype=chatgpt.markdown
    setlocal nospell
    setlocal winfixwidth

    if &equalalways && get(g:, 'vim_chatgpt_autoresize', 1)
        " Cause other windows to resize
        set noequalalways
        let prev_ead = &eadirection
        set eadirection=hor
        set equalalways
        let &eadirection=prev_ead
    endif

    exec "silent" "file" chat#bufname("Chat")

    let outwin = win_getid()
    let chat_job = chat#start(bufnr(), a:model)

    let w:chat_output = 1
    let w:outwin = outwin
    let w:chat_job = chat_job

    16split
    noswapfile hide enew
    setlocal buftype=nofile
    setlocal bufhidden=wipe
    setlocal nobuflisted
    setlocal filetype=chatgpt.markdown
    setlocal winfixheight

    exec "silent" "file" chat#bufname("Input")

    nmap <buffer> <CR> <Cmd>call chat#sendbuf()<CR>
    let w:outwin = outwin
    let w:chat_job = chat_job
endfun

fun chat#start(outbuf, model='')
    let model = a:model

    if model == '' && exists('g:vim_chatgpt_model')
        let model = g:vim_chatgpt_model
    endif

    let args = model != '' ? ['--model', model] : []

    return job_start([g:vim_chatgpt_binary, '--wrap', '80'] + args, #{
    \  mode: 'raw',
    \  in_io: 'pipe',
    \  out_io: 'pipe',
    \  err_io: 'pipe',
    \  out_cb: 'chat#output',
    \  err_cb: 'chat#error',
    \  exit_cb: 'chat#exit',
    \})
endfun

fun chat#bufname(name)
    if bufnr(a:name) < 0
        return a:name
    endif

    let x = 2

    while bufnr(a:name..' '..x) >= 0
        let x += 1
    endwhile

    return a:name..' '..x
endfun

fun chat#stop(job)
    if job_status(a:job) == "run"
        call job_stop(a:job)
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
    call chat#send(md)
endfun

fun chat#output(ch, text)
    let winid = chat#getoutwin(ch_getjob(a:ch))

    if winid < 0
        " TODO: error
        return
    endif

    let outbuf = getwininfo(winid)[0].bufnr
    let newlines = split(a:text, "\n", 1)

    if len(newlines) > 0
        let newlines[0] = getbufoneline(outbuf, '$')..newlines[0]

        call setbufline(outbuf, '$', newlines)

        if win_getid() != winid
            call win_execute(winid, 'normal G', 1)
        endif
    endif
endfun

fun chat#getoutwin(job)
    for winfo in getwininfo()
        if get(winfo.variables, 'chat_job', '') == a:job && get(winfo.variables, 'chat_output', 0)
            return winfo.winid
        endif
    endfor

    return -1
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

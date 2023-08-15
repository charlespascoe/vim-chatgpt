fun chat#edit#getinput(range, line1, line2)
    if a:range == 0
        return ""
    endif

    return join(getline(a:line1, a:line2), "\n")
endfun

fun chat#edit#new(range, line1, line2, model='')
    let input = chat#edit#getinput(a:range, a:line1, a:line2)

    vsplit
    enew

    let outwin = win_getid()
    let outbuf = bufnr()
    " let chat_job = chat#start(bufnr(), a:model)

    let w:chat_output = 1
    " let w:outwin = outwin
    " let w:chat_job = chat_job

    16split
    noswapfile hide enew
    setlocal buftype=nofile
    setlocal bufhidden=wipe
    setlocal nobuflisted
    setlocal filetype=chatgpt.markdown
    setlocal winfixheight

    exec "silent" "file" chat#bufname("Prompt")

    py3 import mdjoin

    nmap <buffer> <CR> <Cmd>call chat#edit#run()<CR>
    let w:input = input
    let w:model = a:model
    let w:outwin = outwin
    let w:outbuf = outbuf
    " let w:chat_job = chat_job
endfun

fun chat#edit#run()
    let md = py3eval('mdjoin.join()')
    let input = w:input
    let outbuf = w:outbuf
    let outwin = w:outwin
    let model = get(w:, 'model', '')

    q

    call win_gotoid(outwin)

    if model == '' && exists('g:vim_chatgpt_model')
        let model = g:vim_chatgpt_model
    endif

    let args = []

    if model != ''
        let args += ['--model', model]
    endif

    let job = job_start([g:vim_chatgpt_binary, 'edit', md] + args, #{
    \  mode:           'raw',
    \  in_io:          'pipe',
    \  out_io:         'pipe',
    \  err_io:         'pipe',
    \  out_modifiable: 1,
    \  out_cb:         'chat#output',
    \  err_cb:         'chat#error',
    \  exit_cb:        'chat#edit#exit',
    \})

    let w:chat_job = job

    let job_chan = job_getchannel(job)

    call ch_sendraw(job_chan, input)
    call ch_close_in(job_chan)
endfun

fun chat#edit#exit(job, status)
    if a:status >= 0
        echom "Exit:" a:status
    else
        echom "Done"
    endif
endfun

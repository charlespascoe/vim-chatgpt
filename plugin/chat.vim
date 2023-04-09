if !exists('g:vim_chatgpt_binary')
    let g:vim_chatgpt_binary = fnamemodify(resolve(expand('<sfile>:p')), ':h:h')..'/vim-chatgpt'
endif

command! Chat call chat#new()

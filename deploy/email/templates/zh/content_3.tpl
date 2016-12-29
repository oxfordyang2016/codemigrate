{{ define "username" }} 您好，{{.TargetUser.Username}}: {{end}}
{{ define "content"}} 您发送的包裹"{{.PkgEvent.Pkg.Title}}", 接收者"{{.PkgEvent.User.Username}}"开始下载{{end}}
{{ define "detail_label"}}详情:{{end}}
{{ define "detail"}} 
    <p><span class="title">包裹大小:</span> {{ .PkgEvent.Pkg.HumanSize }}B</p>
    <p><span class="title">文件个数:</span> {{ .PkgEvent.Pkg.NumFiles }}</p>
    <p><span class="title">开始时间:</span> 北京时间 {{ DatetimeFormat "zh" .PkgEvent.Time }}</p>
{{end}}

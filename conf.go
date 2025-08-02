package xgm
type UserConfig struct {
    Verbose bool `json:"verbose"`
    Terminal string `json:"terminal"`
    DbPath string `json:"db_path"`
    WallpaperPath string `json:"wallpaper_path"`
    WallpaperDefault string `json:"wallpaper_default"`
    PinyinDB string `json:"pinyin_db"`
    PinyinUserDB string `json:"pinyin_user_db"`
    Exts map[string][]string `json:"extensions"`
    SpotlightAction map[string][]string `json:"spotlight_action"`
    BarCells []struct {
        Position int `json:"position"`
        Len int `json:"len,omitempty"`
        MinLen int `json:"min_len,omitempty"`
        Name string `json:"name,omitempty"`
        Glyph string `json:"glyph,omitempty"`
        BgColor string `json:"bg,omitempty"`
        FgColor string `json:"fg,omitempty"`
    } `json:"bar"`
    Latex []string `json:"latex"`
    WidgetColors map[string]string `json:"widget_color"`
    FileColor map[string]string `json:"file_color"`
    FileIcon map[string]string `json:"file_icon"`
    WindowConfigs map[string][4]float64 `json:"window_configs"`
    MaxWorkspaceNameLen int `json:"max_workspace_name_len"`
    WorkspaceIcon string `json:"workspace_icon"`
    TermCols int `json:"term_cols"`
    TermRows int `json:"term_rows"`
    Proxy string `json:"proxy"`
    ForcedTilingClasses []string `json:"forced_tiling_classes"`
    KeyBindings map[string][]string `json:"key_bindings"`
	SpotlightInclude []string `json:"spotlight_include"`
    SpotlightExclude []string `json:"spotlight_exclude"`
    Terminals map[string]struct {
        Args []string `json:"args"`
        RequiresExecutable bool `json:"requires_executable"`
    } `json:"terminals"`
    Paste [2][]string `json:"paste"`
    AppBase []string `json:"app_base"`
    Apps map[string]struct {
        Image string `json:"image"`
        Entrypoint string `json:"entrypoint"`
        Volumes []string `json:"volumes,omitempty"`
        Devices []string `json:"devices,omitempty"`
        Environment []string `json:"environment,omitempty"`
        Flags []string `json:"flags,omitempty"`
        Extensions []string `json:"extensions,omitempty"`
        WorkDir string `json:"workDir"`
    } `json:"apps"`
}
var Conf UserConfig 

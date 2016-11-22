---
title: 'プログラマ用'

layout: nil
---

こ↑こ↓みて、どうぞ

http://hitkey.nekokan.dyndns.info/cmdsJP.htm#BMS

ヘッダフィールド
```
#PLAYER :プレイサイド、必ず 1 指定(どうでもいい)

#GENRE :ジャンル

#TITLE :タイトル

#ARTIST :作曲者

#BPM :初期 BPM

#PLAYLEVEL :曲ごとの難易度(とりあえず 1~10)

#RANK :大雑把な難易度(1 が easy,2 が normal,3 が difficulty)

#WAV01 :曲データ

#WAV02 :ショット音(通常ノーツ)

#WAV03 :ショット音(光るノーツ)
```

メインデータフィールド

`#xxxoo:~`

で定義

xxx が小節番号

oo はチャンネル番号

~は実際の譜面配置

## チャンネル番号について

- 11~13

 - 右レーン番号、下から 11,12,13 となります

 - 配置番号 02 が通常ノーツ、03 が光るノーツ

- 09

 - テンポ変更(ソフラン用)

 - 配置数字がそのまま変更後の BPM になる
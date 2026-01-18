
## 📘 Eğitim: Çok Modüllü Çalışma Alanlarıyla Başlangıç

## 📑 İçindekiler

* Ön koşullar
* Kodunuz için bir modül oluşturun
* Çalışma alanını oluşturun
* `golang.org/x/example/hello` modülünü indirip değiştirin
* Çalışma alanları hakkında daha fazlasını öğrenin

Bu eğitim, Go’da *çok modüllü çalışma alanlarının* temellerini tanıtır. Çok modüllü çalışma alanlarıyla, Go komutuna aynı anda birden fazla modülde kod yazdığınızı söyleyebilir ve bu modüllerdeki kodu kolayca derleyip çalıştırabilirsiniz.

Bu eğitimde, paylaşılan bir *çok modüllü çalışma alanı* içinde iki modül oluşturacak, bu modüller arasında değişiklikler yapacak ve bu değişikliklerin bir derlemede sonuçlarını göreceksiniz.

Not: Diğer eğitimler için bkz. Tutorials.

## ✅ Ön koşullar¶

* Go 1.18 veya daha yenisinin yüklü olması.
* Kodunuzu düzenlemek için bir araç. Elinizdeki herhangi bir metin düzenleyici işinizi görür.
* Bir komut terminali. Go; Linux ve Mac’te herhangi bir terminalde, Windows’ta PowerShell veya cmd üzerinde iyi çalışır.

Bu eğitim *go1.18* veya daha yenisini gerektirir. `go.dev/dl` bağlantılarını kullanarak Go 1.18 veya daha yenisini yüklediğinizden emin olun.

## 🧩 Kodunuz için bir modül oluşturun¶

Başlamak için, yazacağınız kod için bir modül oluşturun.

Bir komut istemi açın ve ev dizininize geçin.

Linux veya Mac’te:

---
```bash
$ cd
```
---

Windows’ta:

---
```cmd
C:\> cd %HOMEPATH%
```
---

Eğitimin geri kalanında, istem olarak `$` gösterilecektir. Kullandığınız komutlar Windows’ta da çalışacaktır.

Komut isteminden, kodunuz için `workspace` adlı bir dizin oluşturun.

---
```bash
$ mkdir workspace
$ cd workspace
```
---

### Initialize the module

Örneğimiz, `golang.org/x/example` modülüne bağımlı olacak yeni bir `hello` modülü oluşturacaktır.

`hello` modülünü oluşturun:

---
```bash
$ mkdir hello
$ cd hello
$ go mod init example.com/hello
go: creating new go.mod: module example.com/hello
```
---

`go get` kullanarak `golang.org/x/example/hello/reverse` paketine bir bağımlılık ekleyin.

---
```bash
$ go get golang.org/x/example/hello/reverse
```
---

`hello` dizininde aşağıdaki içerikle `hello.go` oluşturun:

---
```go
package main

import (
    "fmt"

    "golang.org/x/example/hello/reverse"
)

func main() {
    fmt.Println(reverse.String("Hello"))
}
```
---

Şimdi `hello` programını çalıştırın:

---
```bash
$ go run .
olleH
```
---

## 🧰 Çalışma alanını oluşturun¶

Bu adımda, modülü içeren bir çalışma alanını belirtmek için bir `go.work` dosyası oluşturacağız.

### Initialize the workspace¶

`workspace` dizininde şunu çalıştırın:

---
```bash
$ go work init ./hello
```
---

`go work init` komutu, Go’ya `./hello` dizinindeki modülleri içeren bir çalışma alanı için bir `go.work` dosyası oluşturmasını söyler.

Go komutu aşağıdakine benzer bir `go.work` dosyası üretir:

---
```go
go 1.18

use ./hello
```
---

`go.work` dosyası, `go.mod` ile benzer bir söz dizimine sahiptir.

`go` yönergesi, Go’ya dosyanın hangi Go sürümüyle yorumlanması gerektiğini söyler. `go.mod` dosyasındaki `go` yönergesine benzer.

`use` yönergesi, Go’ya `hello` dizinindeki modülün bir derleme yaparken ana modüller ( *main modules* ) olması gerektiğini söyler.

Dolayısıyla `workspace` altındaki herhangi bir alt dizinde modül etkin olacaktır.

### Programı workspace dizininde çalıştırın¶

`workspace` dizininde şunu çalıştırın:

---
```bash
$ go run ./hello
olleH
```
---

Go komutu, çalışma alanındaki tüm modülleri ana modüller olarak dahil eder. Bu, modülün dışında olsanız bile modüldeki bir pakete başvurabilmemizi sağlar. Çalışma alanının dışında `go run` komutunu çalıştırmak hata verecektir; çünkü Go komutu hangi modülleri kullanacağını bilemez.

Sonraki adımda, `golang.org/x/example/hello` modülünün yerel bir kopyasını çalışma alanına ekleyeceğiz. Bu modül, `go.googlesource.com/example` Git deposunun bir alt dizininde tutulur. Ardından `reverse` paketine, `String` yerine kullanabileceğimiz yeni bir fonksiyon ekleyeceğiz.

## 📥 `golang.org/x/example/hello` modülünü indirip değiştirin¶

Bu adımda, `golang.org/x/example/hello` modülünü içeren Git deposunun bir kopyasını indirecek, çalışma alanına ekleyecek ve ardından `hello` programından kullanacağımız yeni bir fonksiyon ekleyeceğiz.

### Depoyu klonlayın

`workspace` dizininden, depoyu klonlamak için `git` komutunu çalıştırın:

---
```bash
$ git clone https://github.com/1DeliDolu/pehlione_go.git
Cloning into 'example'...
remote: Total 165 (delta 27), reused 165 (delta 27)
Receiving objects: 100% (165/165), 434.18 KiB | 1022.00 KiB/s, done.
Resolving deltas: 100% (27/27), done.
```
---

### Modülü çalışma alanına ekleyin

Git deposu az önce `./example` içine alındı. `golang.org/x/example/hello` modülünün kaynak kodu `./example/hello` içindedir. Bunu çalışma alanına ekleyin:

---
```bash
$ go work use ./example/hello
```
---

`go work use` komutu, `go.work` dosyasına yeni bir modül ekler. Artık şu şekilde görünecektir:

---
```go
go 1.18

use (
    ./hello
    ./example/hello
)
```
---

Çalışma alanı artık hem `example.com/hello` modülünü hem de `golang.org/x/example/hello` modülünü içerir; bu modül de `golang.org/x/example/hello/reverse` paketini sağlar.

Bu, `reverse` paketinin, `go get` komutuyla indirdiğimiz modül önbelleğindeki ( *module cache* ) sürümü yerine, bizim kopyamızdaki yeni yazacağımız kodu kullanmamıza izin verir.

### Yeni fonksiyonu ekleyin

`golang.org/x/example/hello/reverse` paketine, bir sayıyı tersine çevirmek için yeni bir fonksiyon ekleyeceğiz.

`workspace/example/hello/reverse` dizininde `int.go` adlı yeni bir dosya oluşturun ve aşağıdaki içeriği ekleyin:

---
```go
package reverse

import "strconv"

// Int returns the decimal reversal of the integer i.
func Int(i int) int {
    i, _ = strconv.Atoi(String(strconv.Itoa(i)))
    return i
}
```
---

### Hello programını fonksiyonu kullanacak şekilde değiştirin

`workspace/hello/hello.go` içeriğini aşağıdaki gibi olacak şekilde değiştirin:

---
```go
package main

import (
    "fmt"

    "golang.org/x/example/hello/reverse"
)

func main() {
    fmt.Println(reverse.String("Hello"), reverse.Int(24601))
}
```
---

## 🚀 Kodu çalışma alanında çalıştırın¶

`workspace` dizininden şunu çalıştırın:

---
```bash
$ go run ./hello
olleH 10642
```
---

Go komutu, komut satırında belirtilen `example.com/hello` modülünü, `go.work` dosyasında belirtilen `hello` dizininde bulur; benzer şekilde `golang.org/x/example/hello/reverse` içe aktarımını da `go.work` dosyasını kullanarak çözümler.

`go.work`, birden fazla modül üzerinde çalışmak için `replace` yönergeleri eklemek yerine kullanılabilir.

İki modül aynı çalışma alanında olduğundan, bir modülde değişiklik yapıp diğerinde kullanmak kolaydır.

## 🔮 Gelecekteki adım¶

Şimdi, bu modülleri düzgün şekilde yayınlamak için `golang.org/x/example/hello` modülünün bir sürümünü yayınlamamız gerekir; örneğin `v0.1.0`. Bu genellikle modülün sürüm kontrol deposunda bir commit’i etiketleyerek ( *tagging* ) yapılır. Daha fazla ayrıntı için modül sürüm yayınlama iş akışı dokümantasyonuna bakın. Yayınlama yapıldıktan sonra, `hello/go.mod` içindeki `golang.org/x/example/hello` gereksinimini artırabiliriz:

---
```bash
cd hello
go get golang.org/x/example/hello@v0.1.0
```
---

Böylece Go komutu, modülleri çalışma alanı dışında düzgün şekilde çözümleyebilir.

## 📚 Çalışma alanları hakkında daha fazlasını öğrenin¶

Go komutunun, az önce gördüğümüz `go work init` dışında çalışma alanlarıyla çalışmak için birkaç alt komutu daha vardır:

* `go work use [-r] [dir]` `dir` için `go.work` dosyasına bir `use` yönergesi ekler (varsa) ve argüman olarak verilen dizin yoksa `use` girdisini kaldırır. `-r` bayrağı, `dir` alt dizinlerini özyinelemeli olarak inceler.
* `go work edit` `go.work` dosyasını, `go mod edit` benzeri şekilde düzenler
* `go work sync` çalışma alanının derleme listesindeki bağımlılıkları her bir çalışma alanı modülüne senkronlar.

Çalışma alanları ve `go.work` dosyaları hakkında daha fazla ayrıntı için Go Modules Reference içindeki Workspaces bölümüne bakın.

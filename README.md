## Grower

### Задумки

- Партиции хранить в директориях топиков
- Партиции разделить на сегменты (по весу или по количеству сообщений), каждый сегмент хранить в директории партиции
- Подумать и реализовать политику хранения сегментов, например, удалять все сегменты старше 5 дней или когда сумма сегментов превышает 100k или размер превышает 1ГБ удалять первые пришедшие (FIFO)
- Сегменты нумеровать так 0-10 сообщений 00000000000.growerlog и 10-20 сообщений 00000000010.growerlog
- Добавить индексные файлы к сегментам с офсетами и позициями байтов, формат сегмента:

```log
offset | position
1      | 0
2      | 45
```

- Для быстрого поиска и чтения использовать индексный файл по аналогии с логами (00000000000.growerindex, 00000000010.growerindex) с бинарным поиском, затем делать побайтовое смещение типа Seek
- При записи нужно будет сохранять смещение каждой записи (начало смещения)
- Подумать над gRPC для клиентов
- zero copy disk -> socket

### Общая схема работы

```shell
// create two topics with 3 and 2 partitions
$ create topics -> rainbow(3), sanbox(2)

// producer 18 messages write to rainbow topic
$ producer -> (rainbow, 'Hello, Kitty 0!')
$ producer -> (rainbow, 'Hello, Kitty 1!')
$ producer -> (rainbow, 'Hello, Kitty 2!')
$ ...
$ producer -> (rainbow, 'Hello, Kitty 10!')

// describe topic in this moment, partition size max is 3 message
$ desc topic rainbow
-- /rainbow
  -- /0
    -- 00.growerlog   ['Hello, Kitty 0!', 'Hello, Kitty 3!', 'Hello, Kitty 6!']
    -- 00.growerindex 
    -- 03.growerlog   ['Hello, Kitty 9!', 'Hello, Kitty 12!', 'Hello, Kitty 15!']
    -- 03.growerindex
  -- /1
    -- 00.growerlog   ['Hello, Kitty 1!', 'Hello, Kitty 4!', 'Hello, Kitty 7!']
    -- 00.growerindex
    -- 03.growerlog   ['Hello, Kitty 10!', 'Hello, Kitty 13!', 'Hello, Kitty 16!']
    -- 03.growerindex
  -- /2
    -- 00.growerlog   ['Hello, Kitty 2!', 'Hello, Kitty 5!', 'Hello, Kitty 8!']
    -- 00.growerindex
    -- 03.growerlog   ['Hello, Kitty 11!', 'Hello, Kitty 14!', 'Hello, Kitty 17!']
    -- 03.growerindex
```

```shell
$ S1 = subscribe -> rainbow
$ S2 = subscribe -> rainbow

// S1 linked partitions [0, 1]
// S2 linked partitions [2]

// batch size 3
$ S1 -> read [ all from 0 and 1 partitions ]
$ S2 -> read [ all from last partition ]
```

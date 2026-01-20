SELECT 
    cbd.cdkey_blank,
    cob.label,
    cbd.status_blank,
    cbd.cdlpuparent_blank_from
FROM public.constructor_blank_display cbd
JOIN public.constructor_original_blank cob ON cbd.cdkey_blank = cob.cdkey_blank
WHERE cob.label ILIKE '%Приложение №2 Приказ МЗ РФ 165н%'
   OR cob.label ILIKE '%Приложение №3 Приказ МЗ РФ 165н%';

-- Отключаем все записи этих бланков (меняем status_blank с 2 на 0) при необходимости можно вернуть с 0 на 2 и они снова будцт в системе TS
UPDATE public.constructor_blank_display
SET status_blank = 0
WHERE cdkey_blank IN (
    SELECT cdkey_blank
    FROM public.constructor_original_blank
    WHERE label ILIKE '%Приложение №2 Приказ МЗ РФ 165н%'
       OR label ILIKE '%Приложение №3 Приказ МЗ РФ 165н%'
);
SELECT 
    cbd.cdkey_blank,
    cob.label,
    cbd.status_blank,
    cbd.cdlpuparent_blank_from
FROM public.constructor_blank_display cbd
JOIN public.constructor_original_blank cob ON cbd.cdkey_blank = cob.cdkey_blank
WHERE cob.label ILIKE '%Приложение №2 Приказ МЗ РФ 165н%'
   OR cob.label ILIKE '%Приложение №3 Приказ МЗ РФ 165н%';

